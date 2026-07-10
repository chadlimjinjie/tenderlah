//! Per-app maintenance flags keyed by a "scope" (e.g. "beta" for web-beta,
//! "portfolio" for web-chad). Read is public; write is admin-only (global
//! "admin" role from the JWT). Port of internal/handlers/maintenance.

use axum::extract::rejection::JsonRejection;
use axum::extract::{Path, State};
use axum::http::StatusCode;
use axum::response::{IntoResponse, Json, Response};
use chrono::{DateTime, Local};
use serde::{Deserialize, Serialize};

use crate::db;
use crate::db::models::serialize_opt_go_time;
use crate::error::{error_json, internal_error};
use crate::middleware::RequireAdmin;
use crate::state::AppState;

#[derive(Serialize)]
pub struct MaintenanceResponse {
    pub enabled: bool,
    pub message: String,
    #[serde(rename = "updatedAt", serialize_with = "serialize_opt_go_time")]
    pub updated_at: Option<DateTime<Local>>,
}

#[derive(Deserialize)]
pub struct SetMaintenanceRequest {
    #[serde(default)]
    pub enabled: bool,
    #[serde(default)]
    pub message: String,
}

/// GET /api/maintenance/{scope} — public.
pub async fn get_scoped(State(state): State<AppState>, Path(scope): Path<String>) -> Response {
    match db::maintenance::get_maintenance_scope(&state.pool, &scope).await {
        Ok(row) => Json(MaintenanceResponse {
            enabled: row.enabled,
            message: row.message,
            updated_at: Some(row.updated_at),
        })
        .into_response(),
        // Scope not yet created — report the safe default of disabled.
        Err(sqlx::Error::RowNotFound) => Json(MaintenanceResponse {
            enabled: false,
            message: String::new(),
            updated_at: None,
        })
        .into_response(),
        Err(err) => internal_error(err),
    }
}

/// PUT /api/maintenance/{scope} — admin only.
pub async fn set_scoped(
    RequireAdmin(_user): RequireAdmin,
    State(state): State<AppState>,
    Path(scope): Path<String>,
    body: Result<Json<SetMaintenanceRequest>, JsonRejection>,
) -> Response {
    let Json(body) = match body {
        Ok(b) => b,
        Err(rej) => return error_json(StatusCode::BAD_REQUEST, rej.to_string()),
    };
    match db::maintenance::upsert_maintenance_scope(&state.pool, &scope, body.enabled, &body.message).await {
        Ok(row) => Json(MaintenanceResponse {
            enabled: row.enabled,
            message: row.message,
            updated_at: Some(row.updated_at),
        })
        .into_response(),
        Err(err) => internal_error(err),
    }
}
