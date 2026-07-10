//! User onboarding queries ported verbatim from db/users/query.sql (sqlc),
//! plus the hand-written embedding accessors from
//! internal/db/user_onboarding_embedding.go (hand-written there because sqlc
//! drops `vector` columns).

use pgvector::Vector;
use sqlx::PgPool;

use super::models::UserOnboarding;

// Column lists below are `SELECT *` minus the vector `embedding` column
// (dropped by sqlc; never returned by the API).

pub async fn get_user_onboarding(pool: &PgPool, user_id: &str) -> Result<UserOnboarding, sqlx::Error> {
    let mut tx = pool.begin().await?;
    let out = sqlx::query_as(
        r#"SELECT "userId", "accountType", "orgName", "businessName", "businessDescription", "industry", "logoUrl", "companySize", "geographicCoverage", "preferredContractSize", "createdAt", "updatedAt" FROM user_onboarding WHERE "userId" = $1"#,
    )
    .bind(user_id)
    .persistent(false)
    .fetch_one(&mut *tx)
    .await?;
    tx.commit().await?;
    Ok(out)
}

pub struct UpsertUserOnboardingParams<'a> {
    pub user_id: &'a str,
    pub account_type: &'a str,
    pub org_name: &'a str,
    pub business_name: Option<&'a str>,
    pub business_description: Option<&'a str>,
    pub industry: &'a str,
    pub company_size: Option<&'a str>,
    pub geographic_coverage: Option<&'a str>,
    pub preferred_contract_size: Option<&'a str>,
}

pub async fn upsert_user_onboarding(
    pool: &PgPool,
    p: UpsertUserOnboardingParams<'_>,
) -> Result<UserOnboarding, sqlx::Error> {
    let mut tx = pool.begin().await?;
    let out = sqlx::query_as(
        r#"INSERT INTO user_onboarding ("userId", "accountType", "orgName", "businessName", "businessDescription", "industry", "companySize", "geographicCoverage", "preferredContractSize", "createdAt", "updatedAt")
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT ("userId") DO UPDATE
SET "accountType" = $2, "orgName" = $3, "businessName" = $4, "businessDescription" = $5, "industry" = $6,
    "companySize" = $7, "geographicCoverage" = $8, "preferredContractSize" = $9, "updatedAt" = CURRENT_TIMESTAMP
RETURNING "userId", "accountType", "orgName", "businessName", "businessDescription", "industry", "logoUrl", "companySize", "geographicCoverage", "preferredContractSize", "createdAt", "updatedAt""#,
    )
    .bind(p.user_id)
    .bind(p.account_type)
    .bind(p.org_name)
    .bind(p.business_name)
    .bind(p.business_description)
    .bind(p.industry)
    .bind(p.company_size)
    .bind(p.geographic_coverage)
    .bind(p.preferred_contract_size)
    .persistent(false)
    .fetch_one(&mut *tx)
    .await?;
    tx.commit().await?;
    Ok(out)
}

pub async fn set_user_onboarding_logo(
    pool: &PgPool,
    user_id: &str,
    logo_url: &str,
) -> Result<UserOnboarding, sqlx::Error> {
    let mut tx = pool.begin().await?;
    let out = sqlx::query_as(
        r#"UPDATE user_onboarding SET "logoUrl" = $2, "updatedAt" = CURRENT_TIMESTAMP
WHERE "userId" = $1
RETURNING "userId", "accountType", "orgName", "businessName", "businessDescription", "industry", "logoUrl", "companySize", "geographicCoverage", "preferredContractSize", "createdAt", "updatedAt""#,
    )
    .bind(user_id)
    .bind(logo_url)
    .persistent(false)
    .fetch_one(&mut *tx)
    .await?;
    tx.commit().await?;
    Ok(out)
}

/// Stores the profile embedding for a user. The vector is an embed-service
/// (potion-base-8M, 256-dim) embedding of the onboarding profile, in the same
/// space as tender.embedding so relevance ranking can cosine-compare the two.
pub async fn set_user_onboarding_embedding(pool: &PgPool, user_id: &str, vec: Vector) -> Result<(), sqlx::Error> {
    let mut tx = pool.begin().await?;
    sqlx::query(r#"UPDATE user_onboarding SET "embedding" = $2 WHERE "userId" = $1"#)
        .bind(user_id)
        .bind(vec)
        .persistent(false)
        .execute(&mut *tx)
        .await?;
    tx.commit().await
}

/// Returns the stored profile embedding, or None when the user has no
/// onboarding row or the column is still NULL (onboarded before embeddings
/// were precomputed). Read as text and parsed, matching the Go path.
pub async fn get_user_onboarding_embedding(pool: &PgPool, user_id: &str) -> Result<Option<Vector>, sqlx::Error> {
    let mut tx = pool.begin().await?;
    let s: Option<Option<String>> =
        sqlx::query_scalar(r#"SELECT "embedding"::text FROM user_onboarding WHERE "userId" = $1"#)
            .bind(user_id)
            .persistent(false)
            .fetch_optional(&mut *tx)
            .await?;
    tx.commit().await?;
    let Some(Some(text)) = s else { return Ok(None) };
    Ok(parse_vector_text(&text))
}

/// Parses pgvector's text form "[0.1,0.2,...]" into a Vector. Malformed text
/// yields None (the Go version surfaces a scan error; either way the caller
/// logs and falls back to recency sort).
fn parse_vector_text(s: &str) -> Option<Vector> {
    let inner = s.trim().strip_prefix('[')?.strip_suffix(']')?;
    let mut vals = Vec::new();
    for part in inner.split(',') {
        vals.push(part.trim().parse::<f32>().ok()?);
    }
    Some(Vector::from(vals))
}
