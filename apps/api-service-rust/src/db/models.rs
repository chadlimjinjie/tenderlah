//! Row structs mirroring the sqlc-generated models in internal/db/models.go.
//! Serde field order and names must match the Go structs exactly — the JSON
//! shapes are the frozen contract with apps/web-beta.

use chrono::{DateTime, Local, NaiveDate, Offset, Timelike};
use serde::{Serialize, Serializer};

/// Serializes a Postgres `date` the way Go's *time.Time does (pgx reads dates
/// as midnight UTC, so Go emits e.g. "2026-05-01T00:00:00Z", not "2026-05-01").
pub fn serialize_date_as_rfc3339<S: Serializer>(d: &Option<NaiveDate>, s: S) -> Result<S::Ok, S::Error> {
    match d {
        Some(d) => s.serialize_str(&format!("{}T00:00:00Z", d.format("%Y-%m-%d"))),
        None => s.serialize_none(),
    }
}

/// Formats a timestamp exactly like Go's time.Time JSON marshalling
/// (RFC3339Nano): local offset (Z when UTC), fractional seconds with ALL
/// trailing zeros trimmed and no dot when zero. chrono's own serializer pads
/// the fraction to 3/6/9 digits, which would diff against the Go service.
pub fn format_go_rfc3339nano(dt: &DateTime<Local>) -> String {
    let mut out = dt.format("%Y-%m-%dT%H:%M:%S").to_string();
    let nanos = dt.nanosecond() % 1_000_000_000;
    if nanos > 0 {
        let mut frac = format!("{nanos:09}");
        while frac.ends_with('0') {
            frac.pop();
        }
        out.push('.');
        out.push_str(&frac);
    }
    let offset_secs = dt.offset().fix().local_minus_utc();
    if offset_secs == 0 {
        out.push('Z');
    } else {
        let sign = if offset_secs < 0 { '-' } else { '+' };
        let abs = offset_secs.abs();
        out.push_str(&format!("{sign}{:02}:{:02}", abs / 3600, (abs % 3600) / 60));
    }
    out
}

pub fn serialize_go_time<S: Serializer>(dt: &DateTime<Local>, s: S) -> Result<S::Ok, S::Error> {
    s.serialize_str(&format_go_rfc3339nano(dt))
}

pub fn serialize_opt_go_time<S: Serializer>(dt: &Option<DateTime<Local>>, s: S) -> Result<S::Ok, S::Error> {
    match dt {
        Some(dt) => serialize_go_time(dt, s),
        None => s.serialize_none(),
    }
}

/// `#[sqlx(default)]` on the details/AI columns lets this one struct decode
/// both the full GetTender row and the 15-column search row; unselected fields
/// stay None and serialize as null, exactly like the Go struct's zero values.
#[derive(Debug, Clone, Serialize, sqlx::FromRow)]
#[serde(rename_all = "camelCase")]
pub struct Tender {
    pub id: String,
    pub source: String,
    pub source_id: String,
    pub reference_no: String,
    pub title: String,
    pub link: String,
    pub category: String,
    pub agency: String,
    pub tender_type: String,
    #[serde(serialize_with = "serialize_date_as_rfc3339")]
    pub published_date: Option<NaiveDate>,
    #[serde(serialize_with = "serialize_opt_go_time")]
    pub closing_date: Option<DateTime<Local>>,
    pub status: String,
    pub description: String,
    #[serde(serialize_with = "serialize_go_time")]
    pub first_seen_at: DateTime<Local>,
    #[serde(serialize_with = "serialize_go_time")]
    pub last_seen_at: DateTime<Local>,
    /// Source-specific JSON (contact + items/description); null until scraped.
    #[sqlx(default)]
    pub details: Option<serde_json::Value>,
    #[sqlx(default)]
    #[serde(serialize_with = "serialize_opt_go_time")]
    pub details_fetched_at: Option<DateTime<Local>>,
    /// Generated bid brief (scope/requirements/risk flags); null until generated.
    #[sqlx(default)]
    pub ai_brief: Option<serde_json::Value>,
    #[sqlx(default)]
    #[serde(serialize_with = "serialize_opt_go_time")]
    pub ai_brief_at: Option<DateTime<Local>>,
    /// Generated delivery milestone timeline; null until generated.
    #[sqlx(default)]
    pub ai_delivery_timeline: Option<serde_json::Value>,
    #[sqlx(default)]
    #[serde(serialize_with = "serialize_opt_go_time")]
    pub ai_delivery_timeline_at: Option<DateTime<Local>>,
}

#[derive(Debug, Clone, sqlx::FromRow)]
pub struct UserOnboarding {
    #[sqlx(rename = "userId")]
    #[allow(dead_code)] // decoded for row-shape parity; responses exclude it
    pub user_id: String,
    #[sqlx(rename = "accountType")]
    pub account_type: String,
    #[sqlx(rename = "orgName")]
    pub org_name: String,
    #[sqlx(rename = "businessName")]
    pub business_name: Option<String>,
    #[sqlx(rename = "businessDescription")]
    pub business_description: Option<String>,
    pub industry: String,
    #[sqlx(rename = "logoUrl")]
    pub logo_url: Option<String>,
    #[sqlx(rename = "companySize")]
    pub company_size: Option<String>,
    #[sqlx(rename = "geographicCoverage")]
    pub geographic_coverage: Option<String>,
    #[sqlx(rename = "preferredContractSize")]
    pub preferred_contract_size: Option<String>,
    #[sqlx(rename = "createdAt")]
    pub created_at: DateTime<Local>,
    #[sqlx(rename = "updatedAt")]
    pub updated_at: DateTime<Local>,
}

/// Per-(user, tender) AI fit score; distinct from Tender.ai_brief because fit
/// depends on the viewer's company profile.
#[derive(Debug, Clone, sqlx::FromRow)]
pub struct TenderMatchRow {
    #[sqlx(rename = "userId")]
    #[allow(dead_code)] // decoded for row-shape parity; responses exclude it
    pub user_id: String,
    #[sqlx(rename = "tenderId")]
    #[allow(dead_code)] // decoded for row-shape parity; responses exclude it
    pub tender_id: String,
    pub score: i32,
    /// HIGH | MEDIUM | LOW
    pub priority: String,
    #[sqlx(rename = "fitReasons")]
    pub fit_reasons: serde_json::Value,
    #[sqlx(rename = "missingRequirements")]
    pub missing_requirements: serde_json::Value,
    pub risks: serde_json::Value,
}

#[derive(Debug, Clone, sqlx::FromRow)]
pub struct MaintenanceScopeRow {
    pub enabled: bool,
    pub message: String,
    #[sqlx(rename = "updatedAt")]
    pub updated_at: DateTime<Local>,
}
