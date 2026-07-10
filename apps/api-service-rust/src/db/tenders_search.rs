//! Port of internal/db/tenders_search.go — hand-written (not sqlc) because the
//! WHERE/ORDER BY are conditional on optional filters. Sort columns and
//! directions are restricted to fixed whitelists so they can be concatenated
//! into the query safely; every value is bound, never interpolated.

use pgvector::Vector;
use sqlx::{PgPool, Postgres, QueryBuilder};

use super::models::Tender;

/// GeBIZ is hidden from the public listing pending written DSTA permission
/// (see apps/tender-worker/internal/gebiz/COMPLIANCE.md). Flip
/// GEBIZ_LISTING_ENABLED=true to show it again — no code change needed.
/// Read per-request (not snapshotted at boot), like the Go os.Getenv calls.
pub fn gebiz_listing_enabled() -> bool {
    std::env::var("GEBIZ_LISTING_ENABLED").as_deref() == Ok("true")
}

#[derive(Default)]
pub struct ListTendersSearchParams {
    /// Matched against title/agency/reference_no via ILIKE.
    pub q: String,
    pub status: String,
    /// "open" | "closed" (by closing_date vs now); "" = all.
    pub open_state: String,
    pub source: String,
    pub agency: String,
    /// "closingDate" | "publishedDate" | "title" | "relevance".
    pub sort_by: String,
    /// "asc" | "desc".
    pub sort_dir: String,
    pub favourites_only: bool,
    /// Required when favourites_only is set.
    pub user_id: String,
    /// Signed-in user's onboarding-profile embedding, used when sort_by is "relevance".
    pub relevance_embedding: Option<Vector>,
    pub limit: i64,
    pub offset: i64,
}

/// Parses a pipe-separated multi-select filter value into its non-empty,
/// trimmed parts (delimiter must match the frontend's FilterPicker join —
/// pipe, not comma, because agency names can contain commas).
pub fn split_multi(s: &str) -> Vec<String> {
    s.split('|')
        .map(str::trim)
        .filter(|v| !v.is_empty())
        .map(str::to_string)
        .collect()
}

fn push_where(qb: &mut QueryBuilder<Postgres>, p: &ListTendersSearchParams) {
    let mut has_clause = false;
    let sep = |qb: &mut QueryBuilder<Postgres>, has_clause: &mut bool| {
        qb.push(if *has_clause { " AND " } else { " WHERE " });
        *has_clause = true;
    };

    if !gebiz_listing_enabled() {
        sep(qb, &mut has_clause);
        qb.push("source <> 'gebiz'");
    }
    if !p.q.is_empty() {
        let like = format!("%{}%", p.q);
        sep(qb, &mut has_clause);
        qb.push("(title ILIKE ");
        qb.push_bind(like.clone());
        qb.push(" OR agency ILIKE ");
        qb.push_bind(like.clone());
        qb.push(" OR reference_no ILIKE ");
        qb.push_bind(like);
        qb.push(")");
    }
    if !p.status.is_empty() {
        sep(qb, &mut has_clause);
        qb.push("status = ");
        qb.push_bind(p.status.clone());
    }
    // open/closed is by closing deadline, not the inconsistent source status
    // text: closed = past its closing date; open = upcoming or no deadline set.
    match p.open_state.as_str() {
        "open" => {
            sep(qb, &mut has_clause);
            qb.push("(closing_date IS NULL OR closing_date >= now())");
        }
        "closed" => {
            sep(qb, &mut has_clause);
            qb.push("closing_date < now()");
        }
        _ => {}
    }
    // source/agency accept a pipe-separated list (e.g. "ahtc|mandai|nus") → match any.
    let sources = split_multi(&p.source);
    if !sources.is_empty() {
        sep(qb, &mut has_clause);
        qb.push("source = ANY(");
        qb.push_bind(sources);
        qb.push(")");
    }
    let agencies = split_multi(&p.agency);
    if !agencies.is_empty() {
        sep(qb, &mut has_clause);
        qb.push("agency = ANY(");
        qb.push_bind(agencies);
        qb.push(")");
    }
    if p.favourites_only {
        sep(qb, &mut has_clause);
        qb.push("EXISTS (SELECT 1 FROM tender_favourite tf WHERE tf.tender_id = tender.id AND tf.user_id = ");
        qb.push_bind(p.user_id.clone());
        qb.push(")");
    }
}

/// Builds the ORDER BY clause. "relevance" ranks by cosine distance between
/// each tender's embedding and the caller-supplied profile embedding, falling
/// back to recency when there's no embedding to compare against (anonymous
/// visitors, or users who haven't onboarded / have an empty profile). Under
/// every other sort the embedding still acts as a secondary tie-breaker.
/// The id tie-breaker keeps OFFSET paging stable across queries.
fn push_order_by(qb: &mut QueryBuilder<Postgres>, p: &ListTendersSearchParams) {
    qb.push(" ORDER BY ");
    if p.sort_by == "relevance" {
        match &p.relevance_embedding {
            Some(vec) => {
                qb.push("(CASE WHEN embedding IS NOT NULL THEN embedding <=> ");
                qb.push_bind(vec.clone());
                qb.push(" END) ASC NULLS LAST, published_date DESC NULLS LAST, id DESC");
            }
            None => {
                qb.push("published_date DESC NULLS LAST, id DESC");
            }
        }
        return;
    }

    let col = match p.sort_by.as_str() {
        "closingDate" => "closing_date",
        "publishedDate" => "published_date",
        "title" => "title",
        _ => "published_date",
    };
    let dir = if p.sort_dir == "asc" { "ASC" } else { "DESC" };
    qb.push(format!("{col} {dir} NULLS LAST"));
    if let Some(vec) = &p.relevance_embedding {
        qb.push(", (CASE WHEN embedding IS NOT NULL THEN embedding <=> ");
        qb.push_bind(vec.clone());
        qb.push(" END) ASC NULLS LAST");
    }
    qb.push(format!(", id {dir}"));
}

pub async fn list_tenders_search(pool: &PgPool, p: &ListTendersSearchParams) -> Result<Vec<Tender>, sqlx::Error> {
    let mut qb = QueryBuilder::new(
        "SELECT id, source, source_id, reference_no, title, link, category, agency, tender_type, \
         published_date, closing_date, status, description, first_seen_at, last_seen_at FROM tender",
    );
    push_where(&mut qb, p);
    push_order_by(&mut qb, p);
    qb.push(" LIMIT ");
    qb.push_bind(p.limit);
    qb.push(" OFFSET ");
    qb.push_bind(p.offset);
    let mut tx = pool.begin().await?;
    let out = qb.build_query_as::<Tender>().persistent(false).fetch_all(&mut *tx).await?;
    tx.commit().await?;
    Ok(out)
}

pub async fn count_tenders_search(pool: &PgPool, p: &ListTendersSearchParams) -> Result<i64, sqlx::Error> {
    let mut qb = QueryBuilder::new("SELECT count(*) FROM tender");
    push_where(&mut qb, p);
    let mut tx = pool.begin().await?;
    let out = qb.build_query_scalar::<i64>().persistent(false).fetch_one(&mut *tx).await?;
    tx.commit().await?;
    Ok(out)
}

pub async fn list_distinct_tender_statuses(pool: &PgPool) -> Result<Vec<String>, sqlx::Error> {
    let mut tx = pool.begin().await?;
    let out = sqlx::query_scalar("SELECT DISTINCT status FROM tender WHERE status != '' ORDER BY status")
        .persistent(false)
        .fetch_all(&mut *tx)
        .await?;
    tx.commit().await?;
    Ok(out)
}

pub async fn list_distinct_tender_sources(pool: &PgPool) -> Result<Vec<String>, sqlx::Error> {
    // Query text is built from literals only, never user input.
    let mut query = String::from("SELECT DISTINCT source FROM tender WHERE source != ''");
    if !gebiz_listing_enabled() {
        query.push_str(" AND source <> 'gebiz'");
    }
    query.push_str(" ORDER BY source");
    let mut tx = pool.begin().await?;
    let out = sqlx::query_scalar(sqlx::AssertSqlSafe(query)).persistent(false).fetch_all(&mut *tx).await?;
    tx.commit().await?;
    Ok(out)
}

/// `source` follows the same pipe-separated multi-value convention as the
/// search filters; when non-empty, agencies are scoped to those sources so the
/// frontend's Agency dropdown only lists agencies under the selected Source(s).
pub async fn list_distinct_tender_agencies(pool: &PgPool, source: &str) -> Result<Vec<String>, sqlx::Error> {
    // Query text is built from literals only, never user input (values are bound).
    let mut query = String::from("SELECT DISTINCT agency FROM tender WHERE agency != ''");
    if !gebiz_listing_enabled() {
        query.push_str(" AND source <> 'gebiz'");
    }
    let vals = split_multi(source);
    let mut tx = pool.begin().await?;
    let out = if !vals.is_empty() {
        query.push_str(" AND source = ANY($1)");
        query.push_str(" ORDER BY agency");
        sqlx::query_scalar(sqlx::AssertSqlSafe(query))
            .bind(vals)
            .persistent(false)
            .fetch_all(&mut *tx)
            .await?
    } else {
        query.push_str(" ORDER BY agency");
        sqlx::query_scalar(sqlx::AssertSqlSafe(query)).persistent(false).fetch_all(&mut *tx).await?
    };
    tx.commit().await?;
    Ok(out)
}

#[cfg(test)]
mod tests {
    use super::split_multi;

    #[test]
    fn test_split_multi() {
        assert_eq!(split_multi("ahtc|mandai|nus"), vec!["ahtc", "mandai", "nus"]);
        assert_eq!(split_multi(" a | b "), vec!["a", "b"]);
        assert_eq!(split_multi("||"), Vec::<String>::new());
        assert_eq!(split_multi(""), Vec::<String>::new());
        // Agency names can contain commas — commas must NOT split.
        assert_eq!(split_multi("Ministry of A, B|Other"), vec!["Ministry of A, B", "Other"]);
    }
}
