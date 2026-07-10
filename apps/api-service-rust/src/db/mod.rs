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
/// sessions ("prepared statement already exists").
///
/// Unnamed statements alone are NOT enough, though: sqlx executes every
/// parameterized query as two protocol batches, each ending in its own Sync
/// (Parse+Describe+Sync, wait for ReadyForQuery, then Bind+Execute+Sync).
/// Outside an explicit transaction each Sync ends a pooler "transaction", so
/// under concurrency the pooler can reassign the server connection between
/// the two batches and the Bind hits another client's unnamed statement
/// ("bind message supplies N parameters, but prepared statement \"\" requires
/// M"). Every query in this crate therefore runs inside an explicit
/// pool.begin()..commit() transaction, which pins one server connection for
/// both batches. (pgx's QueryExecModeCacheDescribe doesn't need this because
/// it sends Parse+Bind+Execute+Sync as a single batch.)
pub fn init(database_url: &str) -> Result<PgPool, sqlx::Error> {
    let opts = PgConnectOptions::from_str(database_url)?.statement_cache_capacity(0);
    Ok(PgPoolOptions::new().connect_lazy_with(opts))
}
