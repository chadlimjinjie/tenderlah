//! User onboarding endpoints, port of internal/handlers/users/handler.go.

use axum::extract::rejection::JsonRejection;
use axum::extract::{Multipart, State};
use axum::http::StatusCode;
use axum::response::{IntoResponse, Json, Response};
use chrono::{DateTime, Local};
use pgvector::Vector;
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::db;
use crate::db::models::{serialize_go_time, UserOnboarding};
use crate::db::users::UpsertUserOnboardingParams;
use crate::error::{error_json, internal_error};
use crate::middleware::RequireUser;
use crate::state::AppState;

const MAX_LOGO_SIZE: usize = 5 << 20; // 5MB

fn allowed_logo_ext(content_type: &str) -> Option<&'static str> {
    match content_type {
        "image/jpeg" => Some(".jpg"),
        "image/png" => Some(".png"),
        "image/webp" => Some(".webp"),
        _ => None,
    }
}

/// Response shape — deliberately excludes userId (and the embedding), matching
/// the Go onboardingResponse struct.
#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct OnboardingResponse {
    pub account_type: String,
    pub org_name: String,
    pub business_name: Option<String>,
    pub business_description: Option<String>,
    pub industry: String,
    pub logo_url: Option<String>,
    pub company_size: Option<String>,
    pub geographic_coverage: Option<String>,
    pub preferred_contract_size: Option<String>,
    #[serde(serialize_with = "serialize_go_time")]
    pub created_at: DateTime<Local>,
    #[serde(serialize_with = "serialize_go_time")]
    pub updated_at: DateTime<Local>,
}

fn to_response(u: UserOnboarding) -> OnboardingResponse {
    OnboardingResponse {
        account_type: u.account_type,
        org_name: u.org_name,
        business_name: u.business_name,
        business_description: u.business_description,
        industry: u.industry,
        logo_url: u.logo_url,
        company_size: u.company_size,
        geographic_coverage: u.geographic_coverage,
        preferred_contract_size: u.preferred_contract_size,
        created_at: u.created_at,
        updated_at: u.updated_at,
    }
}

/// GET /api/users/onboarding — auth required.
pub async fn get_onboarding(RequireUser(user): RequireUser, State(state): State<AppState>) -> Response {
    match db::users::get_user_onboarding(&state.pool, &user.id).await {
        Ok(row) => Json(to_response(row)).into_response(),
        Err(sqlx::Error::RowNotFound) => error_json(StatusCode::NOT_FOUND, "Not onboarded"),
        Err(err) => internal_error(err),
    }
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct OnboardingBody {
    #[serde(default)]
    pub account_type: String,
    #[serde(default)]
    pub org_name: String,
    pub business_name: Option<String>,
    pub business_description: Option<String>,
    #[serde(default)]
    pub industry: String,
    pub company_size: Option<String>,
    pub geographic_coverage: Option<String>,
    pub preferred_contract_size: Option<String>,
}

/// POST /api/users/onboarding — auth required. Validation mirrors the Go
/// binding tags (accountType required oneof buyer|supplier|both; orgName and
/// industry required); message text differs from go-playground/validator.
pub async fn upsert_onboarding(
    RequireUser(user): RequireUser,
    State(state): State<AppState>,
    body: Result<Json<OnboardingBody>, JsonRejection>,
) -> Response {
    let Json(body) = match body {
        Ok(b) => b,
        Err(rej) => return error_json(StatusCode::BAD_REQUEST, rej.to_string()),
    };
    if !matches!(body.account_type.as_str(), "buyer" | "supplier" | "both") {
        return error_json(StatusCode::BAD_REQUEST, "accountType must be one of buyer, supplier, both");
    }
    if body.org_name.is_empty() {
        return error_json(StatusCode::BAD_REQUEST, "orgName is required");
    }
    if body.industry.is_empty() {
        return error_json(StatusCode::BAD_REQUEST, "industry is required");
    }

    let row = match db::users::upsert_user_onboarding(
        &state.pool,
        UpsertUserOnboardingParams {
            user_id: &user.id,
            account_type: &body.account_type,
            org_name: &body.org_name,
            business_name: body.business_name.as_deref(),
            business_description: body.business_description.as_deref(),
            industry: &body.industry,
            company_size: body.company_size.as_deref(),
            geographic_coverage: body.geographic_coverage.as_deref(),
            preferred_contract_size: body.preferred_contract_size.as_deref(),
        },
    )
    .await
    {
        Ok(row) => row,
        Err(err) => return internal_error(err),
    };

    // Precompute the relevance embedding so the tenders list never has to
    // embed at request time. Industry weighted ×3 to outweigh the free-text
    // description — must match the read-side weighting in the tenders handler
    // and the Go service byte-for-byte (same vector space as tender.embedding).
    // Best-effort: on failure the column stays NULL and the tenders list
    // self-heals it on the next relevance read.
    let embed_text = format!(
        "{}{}",
        format!("{} ", body.industry).repeat(3),
        body.business_description.as_deref().unwrap_or("")
    );
    match db::embed::embed(&state.embed_service_url, &embed_text).await {
        Err(err) => tracing::error!("users: onboarding embed: {err}"),
        Ok(vec) => {
            if let Err(err) = db::users::set_user_onboarding_embedding(&state.pool, &user.id, Vector::from(vec)).await {
                tracing::error!("users: store onboarding embed: {err}");
            }
        }
    }

    Json(to_response(row)).into_response()
}

/// POST /api/users/onboarding/logo — auth required. Multipart field "file",
/// max 5MB, jpeg/png/webp only; stored in Supabase Storage at
/// {userId}/logo/{uuid}{ext}.
pub async fn upload_logo(
    RequireUser(user): RequireUser,
    State(state): State<AppState>,
    mut multipart: Multipart,
) -> Response {
    // Find the "file" part (Gin's FormFile ignores other fields). A malformed
    // multipart body counts as no file, matching Go's FormFile error path.
    let mut file = None;
    while let Some(field) = multipart.next_field().await.unwrap_or(None) {
        if field.name() != Some("file") {
            continue;
        }
        let content_type = field.content_type().unwrap_or("").to_string();
        match field.bytes().await {
            Ok(data) => file = Some((content_type, data)),
            Err(err) => return error_json(StatusCode::BAD_REQUEST, err.to_string()),
        }
        break;
    }
    let Some((content_type, data)) = file else {
        return error_json(StatusCode::BAD_REQUEST, "file is required");
    };
    if data.len() > MAX_LOGO_SIZE {
        return error_json(StatusCode::BAD_REQUEST, "file exceeds 5MB limit");
    }
    let Some(ext) = allowed_logo_ext(&content_type) else {
        return error_json(StatusCode::BAD_REQUEST, "file must be image/jpeg, image/png, or image/webp");
    };

    let path = format!("{}/logo/{}{}", user.id, Uuid::new_v4(), ext);
    let url = match state.storage.upload(&path, &content_type, data.to_vec()).await {
        Ok(url) => url,
        Err(err) => return internal_error(err),
    };

    match db::users::set_user_onboarding_logo(&state.pool, &user.id, &url).await {
        Ok(row) => Json(to_response(row)).into_response(),
        Err(err) => internal_error(err),
    }
}
