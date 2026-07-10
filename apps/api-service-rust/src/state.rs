use std::sync::Arc;

use dashmap::DashSet;
use sqlx::PgPool;

use crate::auth::JwksVerifier;
use crate::llm::LlmClient;
use crate::storage::SupabaseStorage;

#[derive(Clone)]
pub struct AppState {
    pub pool: PgPool,
    pub jwks: Arc<JwksVerifier>,
    pub llm: Arc<LlmClient>,
    pub storage: Arc<SupabaseStorage>,
    /// apps/embed-service (potion-base-8M) — distinct from llm's cloud chat
    /// endpoint, which serves no embedding models.
    pub embed_service_url: String,
    pub in_flight: Arc<InFlight>,
}

/// Per-process in-flight guards, the port of the Go handler's sync.Maps.
/// In-process only; fine for a single instance. If this ever runs with
/// multiple replicas, swap for a DB-level advisory lock
/// (pg_try_advisory_lock on the tender id) instead.
#[derive(Default)]
pub struct InFlight {
    /// tender id -> guards the GeBIZ detail scrape from concurrent runs.
    pub details: DashSet<String>,
    /// tender id -> guards brief generation from concurrent runs.
    pub briefs: DashSet<String>,
    /// tender id -> guards delivery-timeline generation from concurrent runs.
    pub timelines: DashSet<String>,
    /// "userID:tenderID" -> guards match generation from concurrent runs.
    pub matches: DashSet<String>,
}

/// RAII guard: removing the key on Drop means early returns and errors can't
/// leak an in-flight entry (the Go version uses `defer Delete`).
pub struct InFlightGuard {
    in_flight: Arc<InFlight>,
    set: fn(&InFlight) -> &DashSet<String>,
    key: String,
}

impl Drop for InFlightGuard {
    fn drop(&mut self) {
        (self.set)(&self.in_flight).remove(&self.key);
    }
}

/// Atomically claims `key` in the chosen set. None means another request
/// already holds it (the LoadOrStore "already running" branch).
pub fn try_acquire(
    in_flight: &Arc<InFlight>,
    set: fn(&InFlight) -> &DashSet<String>,
    key: String,
) -> Option<InFlightGuard> {
    if set(in_flight).insert(key.clone()) {
        Some(InFlightGuard { in_flight: Arc::clone(in_flight), set, key })
    } else {
        None
    }
}
