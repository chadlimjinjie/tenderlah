use axum::http::StatusCode;
use axum::response::{IntoResponse, Json, Response};
use serde_json::json;

/// The service's single error body shape: {"error": message} with a status
/// code, mirroring the Go handlers' `c.JSON(status, gin.H{"error": msg})`.
pub fn error_json(status: StatusCode, message: impl Into<String>) -> Response {
    (status, Json(json!({ "error": message.into() }))).into_response()
}

/// 500 with the raw error string — the Go service echoes err.Error() in 500
/// bodies, so this port does the same.
pub fn internal_error(err: impl std::fmt::Display) -> Response {
    error_json(StatusCode::INTERNAL_SERVER_ERROR, err.to_string())
}
