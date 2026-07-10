//! Maintenance-scope queries ported verbatim from db/maintenance/query.sql (sqlc).

use sqlx::PgPool;

use super::models::MaintenanceScopeRow;

pub async fn get_maintenance_scope(pool: &PgPool, scope: &str) -> Result<MaintenanceScopeRow, sqlx::Error> {
    let mut tx = pool.begin().await?;
    let out = sqlx::query_as(r#"SELECT "enabled", "message", "updatedAt" FROM maintenance_scope WHERE "scope" = $1"#)
        .bind(scope)
        .persistent(false)
        .fetch_one(&mut *tx)
        .await?;
    tx.commit().await?;
    Ok(out)
}

pub async fn upsert_maintenance_scope(
    pool: &PgPool,
    scope: &str,
    enabled: bool,
    message: &str,
) -> Result<MaintenanceScopeRow, sqlx::Error> {
    let mut tx = pool.begin().await?;
    let out = sqlx::query_as(
        r#"INSERT INTO maintenance_scope ("scope", "enabled", "message", "updatedAt")
VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
ON CONFLICT ("scope") DO UPDATE SET "enabled" = $2, "message" = $3, "updatedAt" = CURRENT_TIMESTAMP
RETURNING "enabled", "message", "updatedAt""#,
    )
    .bind(scope)
    .bind(enabled)
    .bind(message)
    .persistent(false)
    .fetch_one(&mut *tx)
    .await?;
    tx.commit().await?;
    Ok(out)
}
