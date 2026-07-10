//! Static tender queries ported verbatim from db/tenders/query.sql (sqlc).
//! Column lists are `SELECT *` minus the vector `embedding` column, which sqlc
//! drops and the API never returns.

use sqlx::PgPool;

use super::models::{Tender, TenderMatchRow};

pub async fn get_tender(pool: &PgPool, id: &str) -> Result<Tender, sqlx::Error> {
    sqlx::query_as(
        "SELECT id, source, source_id, reference_no, title, link, category, agency, tender_type, \
         published_date, closing_date, status, description, details, details_fetched_at, \
         ai_brief, ai_brief_at, ai_delivery_timeline, ai_delivery_timeline_at, first_seen_at, last_seen_at \
         FROM tender WHERE id = $1",
    )
    .bind(id)
    .persistent(false)
    .fetch_one(pool)
    .await
}

pub async fn update_tender_details(
    pool: &PgPool,
    id: &str,
    details: &serde_json::Value,
) -> Result<Tender, sqlx::Error> {
    sqlx::query_as(
        "UPDATE tender SET details = $2, details_fetched_at = now() WHERE id = $1 \
         RETURNING id, source, source_id, reference_no, title, link, category, agency, tender_type, \
         published_date, closing_date, status, description, details, details_fetched_at, \
         ai_brief, ai_brief_at, ai_delivery_timeline, ai_delivery_timeline_at, first_seen_at, last_seen_at",
    )
    .bind(id)
    .bind(details)
    .persistent(false)
    .fetch_one(pool)
    .await
}

pub async fn update_tender_brief(
    pool: &PgPool,
    id: &str,
    brief: &serde_json::Value,
) -> Result<Tender, sqlx::Error> {
    sqlx::query_as(
        "UPDATE tender SET ai_brief = $2, ai_brief_at = now() WHERE id = $1 \
         RETURNING id, source, source_id, reference_no, title, link, category, agency, tender_type, \
         published_date, closing_date, status, description, details, details_fetched_at, \
         ai_brief, ai_brief_at, ai_delivery_timeline, ai_delivery_timeline_at, first_seen_at, last_seen_at",
    )
    .bind(id)
    .bind(brief)
    .persistent(false)
    .fetch_one(pool)
    .await
}

pub async fn update_tender_delivery_timeline(
    pool: &PgPool,
    id: &str,
    timeline: &serde_json::Value,
) -> Result<Tender, sqlx::Error> {
    sqlx::query_as(
        "UPDATE tender SET ai_delivery_timeline = $2, ai_delivery_timeline_at = now() WHERE id = $1 \
         RETURNING id, source, source_id, reference_no, title, link, category, agency, tender_type, \
         published_date, closing_date, status, description, details, details_fetched_at, \
         ai_brief, ai_brief_at, ai_delivery_timeline, ai_delivery_timeline_at, first_seen_at, last_seen_at",
    )
    .bind(id)
    .bind(timeline)
    .persistent(false)
    .fetch_one(pool)
    .await
}

pub async fn list_favourite_tender_ids(pool: &PgPool, user_id: &str) -> Result<Vec<String>, sqlx::Error> {
    sqlx::query_scalar("SELECT tender_id FROM tender_favourite WHERE user_id = $1")
        .bind(user_id)
        .persistent(false)
        .fetch_all(pool)
        .await
}

pub async fn create_tender_favourite(pool: &PgPool, user_id: &str, tender_id: &str) -> Result<(), sqlx::Error> {
    sqlx::query("INSERT INTO tender_favourite (user_id, tender_id) VALUES ($1, $2) ON CONFLICT DO NOTHING")
        .bind(user_id)
        .bind(tender_id)
        .persistent(false)
        .execute(pool)
        .await
        .map(|_| ())
}

pub async fn delete_tender_favourite(pool: &PgPool, user_id: &str, tender_id: &str) -> Result<(), sqlx::Error> {
    sqlx::query("DELETE FROM tender_favourite WHERE user_id = $1 AND tender_id = $2")
        .bind(user_id)
        .bind(tender_id)
        .persistent(false)
        .execute(pool)
        .await
        .map(|_| ())
}

pub async fn get_tender_match(pool: &PgPool, user_id: &str, tender_id: &str) -> Result<TenderMatchRow, sqlx::Error> {
    sqlx::query_as(r#"SELECT * FROM tender_match WHERE "userId" = $1 AND "tenderId" = $2"#)
        .bind(user_id)
        .bind(tender_id)
        .persistent(false)
        .fetch_one(pool)
        .await
}

pub struct UpsertTenderMatchParams<'a> {
    pub user_id: &'a str,
    pub tender_id: &'a str,
    pub score: i32,
    pub priority: &'a str,
    pub fit_reasons: &'a serde_json::Value,
    pub missing_requirements: &'a serde_json::Value,
    pub risks: &'a serde_json::Value,
}

pub async fn upsert_tender_match(pool: &PgPool, p: UpsertTenderMatchParams<'_>) -> Result<TenderMatchRow, sqlx::Error> {
    sqlx::query_as(
        r#"INSERT INTO tender_match ("userId", "tenderId", "score", "priority", "fitReasons", "missingRequirements", "risks", "createdAt", "updatedAt")
VALUES ($1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT ("userId", "tenderId") DO UPDATE
SET "score" = $3, "priority" = $4, "fitReasons" = $5, "missingRequirements" = $6, "risks" = $7, "updatedAt" = CURRENT_TIMESTAMP
RETURNING *"#,
    )
    .bind(p.user_id)
    .bind(p.tender_id)
    .bind(p.score)
    .bind(p.priority)
    .bind(p.fit_reasons)
    .bind(p.missing_requirements)
    .bind(p.risks)
    .persistent(false)
    .fetch_one(pool)
    .await
}
