pub mod embed;
pub mod maintenance;
pub mod models;
pub mod tenders;
pub mod tenders_search;
pub mod users;

use std::str::FromStr;

use sqlx::postgres::{PgConnectOptions, PgPoolOptions};
use sqlx::PgPool;

/// Creates the connection pool. Lazy (no dial at startup) to match pgxpool's
/// boot semantics — the Go service starts even when the DB is briefly down.
///
/// Statement caching is disabled (here and via .persistent(false) on every
/// query) because DATABASE_URL points at Supabase's transaction-mode pooler
/// (PgBouncer :6543), where named prepared statements leak across multiplexed
/// sessions ("prepared statement already exists"). Unnamed statements are the
/// sqlx equivalent of the Go service's pgx QueryExecModeCacheDescribe, which
/// is pooler-safe for the same reason.
pub fn init(database_url: &str) -> Result<PgPool, sqlx::Error> {
    let opts = PgConnectOptions::from_str(database_url)?.statement_cache_capacity(0);
    Ok(PgPoolOptions::new().connect_lazy_with(opts))
}
