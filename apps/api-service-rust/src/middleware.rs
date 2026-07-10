//! Port of internal/middleware/auth.go. The global `authenticate` layer is
//! non-aborting — it attaches the verified user to request extensions when a
//! valid Bearer token is present and otherwise does nothing; routes that need
//! auth use the RequireUser/RequireAdmin extractors.

use std::convert::Infallible;
use std::sync::Arc;

use axum::extract::{FromRequestParts, Request, State};
use axum::http::request::Parts;
use axum::http::StatusCode;
use axum::middleware::Next;
use axum::response::Response;

use crate::auth::{parse_bearer, AuthUser};
use crate::error::error_json;
use crate::state::AppState;

pub async fn authenticate(State(state): State<AppState>, mut req: Request, next: Next) -> Response {
    let header = req
        .headers()
        .get(axum::http::header::AUTHORIZATION)
        .and_then(|v| v.to_str().ok())
        .unwrap_or("");
    if let Some(token) = parse_bearer(header) {
        if let Ok(user) = state.jwks.verify_token(token).await {
            req.extensions_mut().insert(Arc::new(user));
        }
    }
    next.run(req).await
}

/// The signed-in user if the request carried a valid token; never rejects.
pub struct MaybeUser(pub Option<Arc<AuthUser>>);

impl<S: Send + Sync> FromRequestParts<S> for MaybeUser {
    type Rejection = Infallible;

    async fn from_request_parts(parts: &mut Parts, _state: &S) -> Result<Self, Self::Rejection> {
        Ok(MaybeUser(parts.extensions.get::<Arc<AuthUser>>().cloned()))
    }
}

/// Rejects with 401 if no authenticated user is present.
pub struct RequireUser(pub Arc<AuthUser>);

impl<S: Send + Sync> FromRequestParts<S> for RequireUser {
    type Rejection = Response;

    async fn from_request_parts(parts: &mut Parts, _state: &S) -> Result<Self, Self::Rejection> {
        parts
            .extensions
            .get::<Arc<AuthUser>>()
            .cloned()
            .map(RequireUser)
            .ok_or_else(unauthorized)
    }
}

/// RequireUser plus a check that the caller's global role is "admin". The role
/// comes from the JWT's "role" claim (set by auth-service) — no DB lookup.
pub struct RequireAdmin(pub Arc<AuthUser>);

impl<S: Send + Sync> FromRequestParts<S> for RequireAdmin {
    type Rejection = Response;

    async fn from_request_parts(parts: &mut Parts, _state: &S) -> Result<Self, Self::Rejection> {
        let user = parts
            .extensions
            .get::<Arc<AuthUser>>()
            .cloned()
            .ok_or_else(unauthorized)?;
        if user.role.as_deref() != Some("admin") {
            return Err(error_json(StatusCode::FORBIDDEN, "Admin access required"));
        }
        Ok(RequireAdmin(user))
    }
}

pub fn unauthorized() -> Response {
    error_json(StatusCode::UNAUTHORIZED, "Unauthorized")
}
