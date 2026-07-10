//! Tender listing/detail/favourites, port of internal/handlers/tenders/handler.go.

use std::collections::HashSet;

use axum::extract::{Path, Query, State};
use axum::http::StatusCode;
use axum::response::{IntoResponse, Json, Response};
use base64::engine::general_purpose::URL_SAFE_NO_PAD;
use base64::Engine;
use serde::{Deserialize, Serialize};

use crate::auth::AuthUser;
use crate::db;
use crate::db::models::Tender;
use crate::db::tenders_search::{gebiz_listing_enabled, ListTendersSearchParams};
use crate::error::{error_json, internal_error};
use crate::middleware::{unauthorized, MaybeUser, RequireUser};
use crate::state::{try_acquire, AppState, InFlight};

use super::tenders_ai::TenderMatchOut;

/// Undoes the URL-safe base64url encoding the frontend applies to ids (every
/// real id is source:source_id, and source_id can contain '/'). A literal id
/// always contains ':' (not in the base64url alphabet) so it fails decode and
/// is used as-is for direct callers.
pub fn decode_tender_id(id: &str) -> String {
    match URL_SAFE_NO_PAD.decode(id) {
        Ok(b) => String::from_utf8(b).unwrap_or_else(|_| id.to_string()),
        Err(_) => id.to_string(),
    }
}

#[derive(Serialize)]
pub struct TenderRow {
    #[serde(flatten)]
    pub tender: Tender,
    #[serde(rename = "isFavourite")]
    pub is_favourite: bool,
    /// Never populated by the current service (parity with the Go struct's
    /// always-nil omitempty field) — kept so the shape is documented.
    #[serde(rename = "match", skip_serializing_if = "Option::is_none")]
    pub match_: Option<TenderMatchOut>,
}

#[derive(Serialize)]
pub struct TenderPage {
    pub items: Vec<TenderRow>,
    pub total: i64,
    pub page: i64,
    #[serde(rename = "pageSize")]
    pub page_size: i64,
}

#[derive(Deserialize, Default)]
pub struct ListQuery {
    page: Option<String>,
    #[serde(rename = "pageSize")]
    page_size: Option<String>,
    q: Option<String>,
    status: Option<String>,
    #[serde(rename = "openState")]
    open_state: Option<String>,
    source: Option<String>,
    agency: Option<String>,
    #[serde(rename = "sortBy")]
    sort_by: Option<String>,
    #[serde(rename = "sortDir")]
    sort_dir: Option<String>,
    #[serde(rename = "favouritesOnly")]
    favourites_only: Option<String>,
}

fn atoi_default(s: Option<&str>, def: i64) -> i64 {
    s.and_then(|v| v.parse::<i64>().ok()).unwrap_or(def)
}

/// GET /api/tenders — public (auth optional; favouritesOnly requires auth).
pub async fn list(
    MaybeUser(user): MaybeUser,
    State(state): State<AppState>,
    Query(query): Query<ListQuery>,
) -> Response {
    let page = atoi_default(query.page.as_deref(), 1).max(1);
    let mut page_size = atoi_default(query.page_size.as_deref(), 50);
    if page_size < 1 {
        page_size = 50;
    }
    page_size = page_size.min(200);

    let favourites_only = query.favourites_only.as_deref() == Some("true");
    let mut user_id = String::new();
    if favourites_only {
        match &user {
            Some(u) => user_id = u.id.clone(),
            None => return unauthorized(),
        }
    }

    // The signed-in user's precomputed onboarding-profile embedding (written on
    // profile save) ranks tenders by cosine distance: it's the primary order for
    // "relevance" sort and a secondary tie-breaker under every other sort, so we
    // load it for any signed-in user. Absent (anonymous visitors, unonboarded
    // users, profiles not yet embedded) it falls back to recency in the sort clause.
    let mut relevance_embedding = None;
    if let Some(u) = &user {
        match db::users::get_user_onboarding_embedding(&state.pool, &u.id).await {
            Err(err) => tracing::error!("tenders: read profile embed: {err}"),
            Ok(stored) => relevance_embedding = stored,
        }
    }

    let params = ListTendersSearchParams {
        q: query.q.unwrap_or_default(),
        status: query.status.unwrap_or_default(),
        open_state: query.open_state.unwrap_or_default(),
        source: query.source.unwrap_or_default(),
        agency: query.agency.unwrap_or_default(),
        sort_by: query.sort_by.unwrap_or_default(),
        sort_dir: query.sort_dir.unwrap_or_default(),
        favourites_only,
        user_id,
        relevance_embedding,
        limit: page_size,
        offset: (page - 1) * page_size,
    };

    let total = match db::tenders_search::count_tenders_search(&state.pool, &params).await {
        Ok(t) => t,
        Err(err) => return internal_error(err),
    };
    let rows = match db::tenders_search::list_tenders_search(&state.pool, &params).await {
        Ok(r) => r,
        Err(err) => return internal_error(err),
    };

    let favourites = favourite_set(&state, user.as_deref()).await;
    let items = rows
        .into_iter()
        .map(|row| TenderRow { is_favourite: favourites.contains(&row.id), tender: row, match_: None })
        .collect();
    Json(TenderPage { items, total, page, page_size }).into_response()
}

/// Returns the signed-in user's favourited tender ids as a set, or an empty
/// set if the request is unauthenticated (list stays public) or the query fails.
async fn favourite_set(state: &AppState, user: Option<&AuthUser>) -> HashSet<String> {
    let Some(user) = user else { return HashSet::new() };
    match db::tenders::list_favourite_tender_ids(&state.pool, &user.id).await {
        Ok(ids) => ids.into_iter().collect(),
        Err(_) => HashSet::new(),
    }
}

/// Serializes an empty Vec as null: the Go handler returns nil slices when a
/// distinct query matches nothing, which marshal as null rather than [].
fn nil_when_empty<S: serde::Serializer>(v: &Vec<String>, s: S) -> Result<S::Ok, S::Error> {
    if v.is_empty() {
        s.serialize_none()
    } else {
        v.serialize(s)
    }
}

#[derive(Serialize)]
pub struct TenderFilters {
    #[serde(serialize_with = "nil_when_empty")]
    pub statuses: Vec<String>,
    #[serde(serialize_with = "nil_when_empty")]
    pub sources: Vec<String>,
    #[serde(serialize_with = "nil_when_empty")]
    pub agencies: Vec<String>,
}

#[derive(Deserialize, Default)]
pub struct FiltersQuery {
    source: Option<String>,
}

/// GET /api/tenders/filters — public.
pub async fn filters(State(state): State<AppState>, Query(query): Query<FiltersQuery>) -> Response {
    let statuses = match db::tenders_search::list_distinct_tender_statuses(&state.pool).await {
        Ok(v) => v,
        Err(err) => return internal_error(err),
    };
    let sources = match db::tenders_search::list_distinct_tender_sources(&state.pool).await {
        Ok(v) => v,
        Err(err) => return internal_error(err),
    };
    let agencies =
        match db::tenders_search::list_distinct_tender_agencies(&state.pool, query.source.as_deref().unwrap_or(""))
            .await
        {
            Ok(v) => v,
            Err(err) => return internal_error(err),
        };
    Json(TenderFilters { statuses, sources, agencies }).into_response()
}

/// GET /api/tenders/{id} — public. AI fields are stripped for anonymous callers.
pub async fn get_one(
    MaybeUser(user): MaybeUser,
    State(state): State<AppState>,
    Path(id): Path<String>,
) -> Response {
    let id = decode_tender_id(&id);
    let mut row = match db::tenders::get_tender(&state.pool, &id).await {
        Ok(r) => r,
        Err(_) => return error_json(StatusCode::NOT_FOUND, "Tender not found"),
    };
    // GeBIZ is hidden from the listing pending DSTA permission (see COMPLIANCE.md);
    // block stale deep-links too. Flip GEBIZ_LISTING_ENABLED=true to surface GeBIZ again.
    if row.source == "gebiz" && !gebiz_listing_enabled() {
        return error_json(StatusCode::NOT_FOUND, "Tender not found");
    }

    let authed = user.is_some();

    // GeBIZ detail isn't in the RSS feed, so on first view of a tender it can be
    // slow. Don't block the response on it — return the cached row immediately
    // and let the frontend poll until it lands. AI brief/timeline generation is
    // triggered separately via POST /{id}/brief and /{id}/delivery-timeline.
    // Flag defaults off pending DSTA written permission to reproduce GeBIZ
    // opportunity-page content; flip GEBIZ_DETAIL_SCRAPE_ENABLED=true once granted.
    let needs_scrape = row.source == "gebiz"
        && row.details_fetched_at.is_none()
        && !row.link.is_empty()
        && std::env::var("GEBIZ_DETAIL_SCRAPE_ENABLED").as_deref() == Ok("true");
    if needs_scrape {
        if let Some(guard) = try_acquire(&state.in_flight, |f: &InFlight| &f.details, row.id.clone()) {
            let state = state.clone();
            let row = row.clone();
            tokio::spawn(async move {
                fetch_details(state, row).await;
                drop(guard);
            });
        }
    }

    if !authed {
        row.ai_brief = None;
        row.ai_brief_at = None;
        row.ai_delivery_timeline = None;
        row.ai_delivery_timeline_at = None;
    }

    let is_favourite = favourite_set(&state, user.as_deref()).await.contains(&row.id);
    Json(TenderRow { tender: row, is_favourite, match_: None }).into_response()
}

/// Scrapes the GeBIZ opportunity page and caches the result. Runs detached
/// from the request (the guard is released when this task finishes).
async fn fetch_details(state: AppState, row: Tender) {
    match crate::gebiz::fetch(&row.link).await {
        Err(err) => tracing::error!("tenders: scrape {}: {err}", row.id),
        Ok(d) => {
            if let Ok(v) = serde_json::to_value(&d) {
                if let Err(err) = db::tenders::update_tender_details(&state.pool, &row.id, &v).await {
                    tracing::error!("tenders: store details {}: {err}", row.id);
                }
            }
        }
    }
}

/// POST /api/tenders/{id}/favourite — auth required.
pub async fn add_favourite(
    RequireUser(user): RequireUser,
    State(state): State<AppState>,
    Path(id): Path<String>,
) -> Response {
    match db::tenders::create_tender_favourite(&state.pool, &user.id, &decode_tender_id(&id)).await {
        Ok(()) => StatusCode::NO_CONTENT.into_response(),
        Err(err) => internal_error(err),
    }
}

/// DELETE /api/tenders/{id}/favourite — auth required.
pub async fn remove_favourite(
    RequireUser(user): RequireUser,
    State(state): State<AppState>,
    Path(id): Path<String>,
) -> Response {
    match db::tenders::delete_tender_favourite(&state.pool, &user.id, &decode_tender_id(&id)).await {
        Ok(()) => StatusCode::NO_CONTENT.into_response(),
        Err(err) => internal_error(err),
    }
}

#[cfg(test)]
mod tests {
    use super::decode_tender_id;

    #[test]
    fn test_decode_tender_id() {
        // base64url("gebiz:EPU000ABC123") round-trips.
        assert_eq!(decode_tender_id("Z2ViaXo6RVBVMDAwQUJDMTIz"), "gebiz:EPU000ABC123");
        // A literal id contains ':' (not base64url) and passes through as-is.
        assert_eq!(decode_tender_id("gebiz:EPU000ABC123"), "gebiz:EPU000ABC123");
        // source_id may contain '/', which the frontend encoding protects.
        assert_eq!(decode_tender_id("c21ydDphL2IvYw"), "smrt:a/b/c");
    }
}
