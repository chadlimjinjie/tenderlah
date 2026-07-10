//! AI brief / delivery-timeline / match endpoints, port of
//! internal/handlers/tenders/{brief,timeline,match}.go. Prompts are copied
//! verbatim — they are part of the product behaviour.

use axum::extract::{Path, State};
use axum::http::StatusCode;
use axum::response::{IntoResponse, Json, Response};
use chrono::{Local, SecondsFormat};
use serde::{Deserialize, Serialize};

use crate::db;
use crate::db::models::{Tender, TenderMatchRow, UserOnboarding};
use crate::db::tenders::UpsertTenderMatchParams;
use crate::error::{error_json, internal_error};
use crate::llm::LlmClient;
use crate::middleware::RequireUser;
use crate::state::{try_acquire, AppState, InFlight};

use super::tenders::decode_tender_id;

/// AI-generated, supplier-agnostic decision brief for one tender. Stored as
/// the tender's ai_brief jsonb and reused for every viewer. Options mirror
/// Go's nil-able slices: a missing section serializes as null, not [].
#[derive(Debug, Default, Serialize, Deserialize)]
pub struct BidBrief {
    /// Plain-English "what they're actually buying".
    #[serde(default)]
    pub summary: String,
    /// Scope as bullet points.
    #[serde(default)]
    pub scope: Option<Vec<String>>,
    /// Eligibility / mandatory certs / registrations.
    #[serde(default)]
    pub requirements: Option<Vec<String>>,
    /// Submission documents the bidder must provide.
    #[serde(default)]
    pub documents: Option<Vec<String>>,
    /// Low | Medium | High.
    #[serde(default)]
    pub complexity: String,
    /// Tight deadline, restrictive specs (possible incumbent), broad scope, etc.
    #[serde(default, rename = "riskFlags")]
    pub risk_flags: Option<Vec<String>>,
}

/// One point on the AI-generated delivery timeline.
#[derive(Debug, Default, Serialize, Deserialize)]
pub struct Milestone {
    /// e.g. "Contract award", "Delivery of Phase 1".
    #[serde(default)]
    pub label: String,
    /// Estimated ISO date (YYYY-MM-DD), always concrete, never relative.
    #[serde(default)]
    pub date: String,
    #[serde(default)]
    pub description: String,
}

/// AI-generated, supplier-agnostic delivery schedule for one tender. Stored as
/// ai_delivery_timeline jsonb. No cap on the number of milestones.
#[derive(Debug, Default, Serialize, Deserialize)]
pub struct DeliveryTimeline {
    #[serde(default)]
    pub milestones: Option<Vec<Milestone>>,
}

/// AI-generated, per-company fit assessment for one tender. Stored per
/// (user, tender) in the tender_match table, not on the tender row.
#[derive(Debug, Default, Serialize, Deserialize)]
pub struct TenderMatchOut {
    /// 0-100.
    #[serde(default)]
    pub score: i64,
    /// HIGH | MEDIUM | LOW.
    #[serde(default)]
    pub priority: String,
    #[serde(default, rename = "fitReasons")]
    pub fit_reasons: Option<Vec<String>>,
    #[serde(default, rename = "missingRequirements")]
    pub missing_requirements: Option<Vec<String>>,
    #[serde(default)]
    pub risks: Option<Vec<String>>,
}

const BRIEF_SYSTEM: &str = "You are a Singapore government procurement analyst helping a supplier decide whether to bid on a tender. \
Read the tender and produce a concise, factual bid brief. Do not invent requirements that are not present. \
Respond with valid JSON only, matching this shape: \
{\"summary\": string, \"scope\": string[], \"requirements\": string[], \"documents\": string[], \"complexity\": \"Low\"|\"Medium\"|\"High\", \"riskFlags\": string[]}. \
riskFlags should call out things a bidder should worry about: a tight closing window, unusually narrow/single-source-looking specifications (possible incumbent advantage), or an unusually broad scope. Use [] when a section has nothing.";

const TIMELINE_SYSTEM: &str = "You are a Singapore government procurement analyst helping a supplier plan delivery against a tender. \
Read the tender and infer a realistic delivery timeline as a sequence of milestones, from contract award through final delivery/acceptance. \
Use the closing date as the basis for the award date when estimating. Every milestone must have a concrete estimated date in YYYY-MM-DD format — never a relative phrase (e.g. \"4 weeks after award\"). \
If the tender doesn't state an exact date for a milestone, estimate one using typical timelines for similar government procurement and express it as a calendar date. \
Do not invent milestones not implied by the tender. Respond with valid JSON only, matching this shape: \
{\"milestones\": [{\"label\": string, \"date\": string, \"description\": string}, ...]}. Order milestones chronologically. Use [] if nothing can be inferred.";

const MATCH_SYSTEM: &str = "You are a Singapore government procurement analyst helping a supplier decide whether to bid on a tender, \
given that supplier's company profile and track record. Read the tender and the company profile and produce a factual fit assessment. \
Do not invent capabilities or requirements that are not present. Score 0-100 reflecting how well the company's profile/products/services/certifications/past projects match what the tender asks for. \
Use this scoring rubric for priority: score >= 70 is HIGH, 40-69 is MEDIUM, below 40 is LOW. \
Respond with valid JSON only, matching this shape: \
{\"score\": number, \"priority\": \"HIGH\"|\"MEDIUM\"|\"LOW\", \"fitReasons\": string[], \"missingRequirements\": string[], \"risks\": string[]}. \
fitReasons should call out concrete reasons the company is/isn't a good fit (industry match, certifications held, relevant past projects). \
missingRequirements should list certifications/registrations/experience the tender seems to require that aren't evidenced in the company profile. \
risks should call out bid risks (tight deadline, narrow specs, broad scope, contract size mismatch). Use [] when a section has nothing.";

fn first_non_empty<'a>(vals: &[&'a str]) -> &'a str {
    vals.iter().find(|v| !v.is_empty()).copied().unwrap_or("")
}

/// Builds the shared tender-description prompt prefix (same lines as the Go
/// brief/timeline/match builders).
fn tender_prompt(t: &Tender) -> String {
    let mut b = String::new();
    b.push_str(&format!("Title: {}\n", t.title));
    b.push_str(&format!("Buyer/Agency: {}\n", first_non_empty(&[&t.agency, &t.source])));
    b.push_str(&format!("Category: {}\n", t.category));
    b.push_str(&format!("Tender type: {}\n", t.tender_type));
    if let Some(closing) = &t.closing_date {
        b.push_str(&format!(
            "Closing date: {} (today is {})\n",
            closing.to_rfc3339_opts(SecondsFormat::Secs, true),
            Local::now().to_rfc3339_opts(SecondsFormat::Secs, false),
        ));
    }
    if !t.description.is_empty() {
        b.push_str(&format!("Description: {}\n", t.description));
    }
    if let Some(details) = &t.details {
        if !details.is_null() {
            b.push_str(&format!("Line items and contact (JSON): {}\n", details));
        }
    }
    b
}

/// Strips an optional ```json fenced block, mirroring the Go parse* helpers.
fn unfence(raw: &str) -> String {
    let mut r = raw.trim();
    let stripped;
    if r.starts_with("```") {
        r = r.strip_prefix("```json").unwrap_or(r);
        r = r.strip_prefix("```").unwrap_or(r);
        stripped = r.trim().strip_suffix("```").unwrap_or(r.trim()).to_string();
        r = &stripped;
        return r.trim().to_string();
    }
    r.to_string()
}

pub fn parse_brief(raw: &str) -> Result<BidBrief, String> {
    serde_json::from_str(&unfence(raw)).map_err(|e| e.to_string())
}

pub fn parse_timeline(raw: &str) -> Result<DeliveryTimeline, String> {
    serde_json::from_str(&unfence(raw)).map_err(|e| e.to_string())
}

pub fn parse_match(raw: &str) -> Result<TenderMatchOut, String> {
    serde_json::from_str(&unfence(raw)).map_err(|e| e.to_string())
}

async fn generate_brief(llm: &LlmClient, t: &Tender) -> Result<serde_json::Value, String> {
    let raw = llm.generate_json(BRIEF_SYSTEM, &tender_prompt(t)).await?;
    let brief = parse_brief(&raw)?;
    serde_json::to_value(&brief).map_err(|e| e.to_string())
}

async fn generate_timeline(llm: &LlmClient, t: &Tender) -> Result<serde_json::Value, String> {
    let raw = llm.generate_json(TIMELINE_SYSTEM, &tender_prompt(t)).await?;
    let timeline = parse_timeline(&raw)?;
    serde_json::to_value(&timeline).map_err(|e| e.to_string())
}

async fn generate_match(
    llm: &LlmClient,
    t: &Tender,
    onboarding: &UserOnboarding,
) -> Result<TenderMatchOut, String> {
    let mut b = tender_prompt(t);
    b.push_str("\nCompany profile:\n");
    b.push_str(&format!("Organisation: {}\n", onboarding.org_name));
    b.push_str(&format!("Industry: {}\n", onboarding.industry));
    b.push_str(&format!("Account type: {}\n", onboarding.account_type));
    if let Some(v) = onboarding.business_description.as_deref().filter(|v| !v.is_empty()) {
        b.push_str(&format!("Business description: {v}\n"));
    }
    if let Some(v) = onboarding.company_size.as_deref().filter(|v| !v.is_empty()) {
        b.push_str(&format!("Company size: {v}\n"));
    }
    if let Some(v) = onboarding.geographic_coverage.as_deref().filter(|v| !v.is_empty()) {
        b.push_str(&format!("Geographic coverage: {v}\n"));
    }
    if let Some(v) = onboarding.preferred_contract_size.as_deref().filter(|v| !v.is_empty()) {
        b.push_str(&format!("Preferred contract size: {v}\n"));
    }

    let raw = llm.generate_json(MATCH_SYSTEM, &b).await?;
    parse_match(&raw)
}

/// Converts the DB row (jsonb columns) into the API response shape, tolerating
/// malformed JSON by leaving the section null (matching the Go best-effort
/// json.Unmarshal into nil-able slices).
fn to_tender_match(m: TenderMatchRow) -> TenderMatchOut {
    TenderMatchOut {
        score: m.score as i64,
        priority: m.priority,
        fit_reasons: serde_json::from_value(m.fit_reasons).ok(),
        missing_requirements: serde_json::from_value(m.missing_requirements).ok(),
        risks: serde_json::from_value(m.risks).ok(),
    }
}

/// POST /api/tenders/{id}/brief — auth required. Returns the cached brief or
/// generates one; 409 when a generation is already in flight.
pub async fn generate_brief_handler(
    RequireUser(_user): RequireUser,
    State(state): State<AppState>,
    Path(id): Path<String>,
) -> Response {
    let id = decode_tender_id(&id);
    let row = match db::tenders::get_tender(&state.pool, &id).await {
        Ok(r) => r,
        Err(_) => return error_json(StatusCode::NOT_FOUND, "Tender not found"),
    };
    if row.ai_brief_at.is_some() {
        return Json(row.ai_brief).into_response();
    }
    if row.title.is_empty() {
        return error_json(StatusCode::BAD_REQUEST, "Tender has no title to generate a brief from");
    }

    let Some(_guard) = try_acquire(&state.in_flight, |f: &InFlight| &f.briefs, row.id.clone()) else {
        return error_json(StatusCode::CONFLICT, "Brief generation already in progress");
    };

    let brief = match generate_brief(&state.llm, &row).await {
        Ok(b) => b,
        Err(err) => return internal_error(err),
    };
    match db::tenders::update_tender_brief(&state.pool, &row.id, &brief).await {
        Ok(updated) => Json(updated.ai_brief).into_response(),
        Err(err) => internal_error(err),
    }
}

/// POST /api/tenders/{id}/delivery-timeline — auth required.
pub async fn generate_timeline_handler(
    RequireUser(_user): RequireUser,
    State(state): State<AppState>,
    Path(id): Path<String>,
) -> Response {
    let id = decode_tender_id(&id);
    let row = match db::tenders::get_tender(&state.pool, &id).await {
        Ok(r) => r,
        Err(_) => return error_json(StatusCode::NOT_FOUND, "Tender not found"),
    };
    if row.ai_delivery_timeline_at.is_some() {
        return Json(row.ai_delivery_timeline).into_response();
    }
    if row.title.is_empty() {
        return error_json(StatusCode::BAD_REQUEST, "Tender has no title to generate a delivery timeline from");
    }

    let Some(_guard) = try_acquire(&state.in_flight, |f: &InFlight| &f.timelines, row.id.clone()) else {
        return error_json(StatusCode::CONFLICT, "Delivery timeline generation already in progress");
    };

    let timeline = match generate_timeline(&state.llm, &row).await {
        Ok(t) => t,
        Err(err) => return internal_error(err),
    };
    match db::tenders::update_tender_delivery_timeline(&state.pool, &row.id, &timeline).await {
        Ok(updated) => Json(updated.ai_delivery_timeline).into_response(),
        Err(err) => internal_error(err),
    }
}

/// POST /api/tenders/{id}/match — auth required. Per-(user, tender) fit score
/// against the caller's company profile; cached in tender_match.
pub async fn generate_match_handler(
    RequireUser(user): RequireUser,
    State(state): State<AppState>,
    Path(id): Path<String>,
) -> Response {
    let id = decode_tender_id(&id);
    let row = match db::tenders::get_tender(&state.pool, &id).await {
        Ok(r) => r,
        Err(_) => return error_json(StatusCode::NOT_FOUND, "Tender not found"),
    };
    if let Ok(cached) = db::tenders::get_tender_match(&state.pool, &user.id, &row.id).await {
        return Json(to_tender_match(cached)).into_response();
    }
    if row.title.is_empty() {
        return error_json(StatusCode::BAD_REQUEST, "Tender has no title to generate a match from");
    }

    let key = format!("{}:{}", user.id, row.id);
    let Some(_guard) = try_acquire(&state.in_flight, |f: &InFlight| &f.matches, key) else {
        return error_json(StatusCode::CONFLICT, "Match generation already in progress");
    };

    let onboarding = match db::users::get_user_onboarding(&state.pool, &user.id).await {
        Ok(o) => o,
        Err(_) => return error_json(StatusCode::BAD_REQUEST, "Complete your business profile first"),
    };
    let matched = match generate_match(&state.llm, &row, &onboarding).await {
        Ok(m) => m,
        Err(err) => return internal_error(err),
    };

    let fit_reasons = serde_json::to_value(&matched.fit_reasons).unwrap_or(serde_json::Value::Null);
    let missing = serde_json::to_value(&matched.missing_requirements).unwrap_or(serde_json::Value::Null);
    let risks = serde_json::to_value(&matched.risks).unwrap_or(serde_json::Value::Null);
    if let Err(err) = db::tenders::upsert_tender_match(
        &state.pool,
        UpsertTenderMatchParams {
            user_id: &user.id,
            tender_id: &row.id,
            score: matched.score as i32,
            priority: &matched.priority,
            fit_reasons: &fit_reasons,
            missing_requirements: &missing,
            risks: &risks,
        },
    )
    .await
    {
        return internal_error(err);
    }
    Json(matched).into_response()
}

#[cfg(test)]
mod tests {
    use super::{parse_brief, parse_timeline};

    /// parse_brief must handle clean JSON and a ```json-fenced response.
    #[test]
    fn test_parse_brief() {
        let cases = [
            r#"{"summary":"Supply of office chairs","scope":["Deliver 100 chairs"],"requirements":["GeBIZ registered"],"documents":["Price schedule"],"complexity":"Low","riskFlags":["Tight 5-day closing"]}"#,
            "```json\n{\"summary\":\"x\",\"scope\":[],\"requirements\":[],\"documents\":[],\"complexity\":\"Medium\",\"riskFlags\":[]}\n```",
        ];
        for (i, raw) in cases.iter().enumerate() {
            let b = parse_brief(raw).unwrap_or_else(|e| panic!("case {i}: {e}"));
            assert!(!b.summary.is_empty() && !b.complexity.is_empty(), "case {i}: empty fields: {b:?}");
        }
        assert!(parse_brief("not json").is_err(), "expected error on garbage input");
    }

    /// parse_timeline must handle clean JSON and a ```json-fenced response,
    /// and must not cap the number of milestones.
    #[test]
    fn test_parse_timeline() {
        let cases = [
            r#"{"milestones":[{"label":"Contract award","date":"2026-07-01","description":"Award notified"},{"label":"Delivery","date":"2026-08-01","description":"Final delivery"}]}"#,
            "```json\n{\"milestones\":[{\"label\":\"Award\",\"date\":\"2026-07-01\",\"description\":\"x\"}]}\n```",
        ];
        for (i, raw) in cases.iter().enumerate() {
            let tl = parse_timeline(raw).unwrap_or_else(|e| panic!("case {i}: {e}"));
            assert!(
                tl.milestones.as_ref().is_some_and(|m| !m.is_empty()),
                "case {i}: expected milestones: {tl:?}"
            );
        }
        assert!(parse_timeline("not json").is_err(), "expected error on garbage input");
    }
}
