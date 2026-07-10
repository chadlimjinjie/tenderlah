//! Midtone API — Rust port of apps/api-service (Go/Gin). Behavior-compatible:
//! same routes, JSON shapes, status codes, auth semantics, and env vars.
//! Intentionally not ported: the Swagger UI at /api/docs (the generated Go
//! docs remain the contract reference).

mod auth;
mod config;
mod db;
mod error;
mod gebiz;
mod handlers;
mod llm;
mod middleware;
mod state;
mod storage;

use std::sync::Arc;

use axum::extract::DefaultBodyLimit;
use axum::response::Json;
use axum::routing::{get, post};
use axum::Router;
use serde_json::json;
use tower_http::catch_panic::CatchPanicLayer;
use tower_http::trace::TraceLayer;

use crate::config::must_env;
use crate::state::{AppState, InFlight};

#[tokio::main]
async fn main() {
    dotenvy::dotenv().ok();
    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| "info,tower_http=info".into()),
        )
        .init();

    let jwks = auth::JwksVerifier::new(&must_env("AUTH_SERVICE_URL", "http://localhost:3100")).await;

    let pool = match db::init(&must_env("DATABASE_URL", "")) {
        Ok(pool) => pool,
        Err(err) => {
            eprintln!("db: {err}");
            std::process::exit(1);
        }
    };

    let llm_client = llm::LlmClient::new(
        must_env("OLLAMA_BASE_URL", "https://ollama.com"),
        std::env::var("OLLAMA_API_KEY").unwrap_or_default(),
        llm::DEFAULT_MODEL.to_string(),
    );

    let supabase_storage = storage::SupabaseStorage::new(
        must_env("SUPABASE_URL", ""),
        must_env("SUPABASE_SERVICE_ROLE_KEY", ""),
        must_env("SUPABASE_STORAGE_BUCKET", "catalogue"),
    );

    let state = AppState {
        pool,
        jwks,
        llm: Arc::new(llm_client),
        storage: Arc::new(supabase_storage),
        embed_service_url: must_env("EMBED_SERVICE_URL", "http://localhost:4107"),
        in_flight: Arc::new(InFlight::default()),
    };

    let app = Router::new()
        .route("/", get(health))
        .route(
            "/api/maintenance/{scope}",
            get(handlers::maintenance::get_scoped).put(handlers::maintenance::set_scoped),
        )
        .route("/api/tenders", get(handlers::tenders::list))
        .route("/api/tenders/filters", get(handlers::tenders::filters))
        .route("/api/tenders/{id}", get(handlers::tenders::get_one))
        .route(
            "/api/tenders/{id}/favourite",
            post(handlers::tenders::add_favourite).delete(handlers::tenders::remove_favourite),
        )
        .route("/api/tenders/{id}/brief", post(handlers::tenders_ai::generate_brief_handler))
        .route(
            "/api/tenders/{id}/delivery-timeline",
            post(handlers::tenders_ai::generate_timeline_handler),
        )
        .route("/api/tenders/{id}/match", post(handlers::tenders_ai::generate_match_handler))
        .route(
            "/api/users/onboarding",
            get(handlers::users::get_onboarding).post(handlers::users::upsert_onboarding),
        )
        .route("/api/users/onboarding/logo", post(handlers::users::upload_logo))
        .layer(axum::middleware::from_fn_with_state(state.clone(), middleware::authenticate))
        .layer(CatchPanicLayer::new())
        .layer(TraceLayer::new_for_http())
        // Gin buffers multipart bodies without a hard cap; axum's 2MB default
        // would reject valid <5MB logo uploads, so raise it (Gin's multipart
        // memory threshold, 32MB, is a reasonable ceiling).
        .layer(DefaultBodyLimit::max(32 * 1024 * 1024))
        .with_state(state);

    let port = must_env("PORT", "4003");
    tracing::info!("Listening on :{port}");
    // Dual-stack listener (IPv6 + IPv4), reproducing Go's net.Listen(":port").
    // Railway's private networking is IPv6-only, so an 0.0.0.0 bind is
    // unreachable from the gateway; set_only_v6(false) makes the behaviour
    // explicit on every platform (Windows defaults to v6-only).
    let listener = match bind_dual_stack(&port) {
        Ok(l) => l,
        Err(err) => {
            eprintln!("listen :{port}: {err}");
            std::process::exit(1);
        }
    };
    if let Err(err) = axum::serve(listener, app).await {
        eprintln!("serve: {err}");
        std::process::exit(1);
    }
}

async fn health() -> Json<serde_json::Value> {
    Json(json!({"ok": true, "service": "midtone-api", "version": "1.0.0"}))
}

fn bind_dual_stack(port: &str) -> std::io::Result<tokio::net::TcpListener> {
    let addr: std::net::SocketAddr = format!("[::]:{port}")
        .parse()
        .map_err(|e| std::io::Error::new(std::io::ErrorKind::InvalidInput, format!("{e}")))?;
    let socket = socket2::Socket::new(socket2::Domain::IPV6, socket2::Type::STREAM, Some(socket2::Protocol::TCP))?;
    socket.set_only_v6(false)?; // accept IPv4 too, like Go
    socket.bind(&addr.into())?;
    socket.listen(1024)?;
    socket.set_nonblocking(true)?;
    tokio::net::TcpListener::from_std(socket.into())
}
