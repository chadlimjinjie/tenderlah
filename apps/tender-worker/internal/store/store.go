// Package store owns the unified `tender` table and the cross-source upsert.
// Each source scraper produces []Record; Upsert writes them keyed by source:source_id.
package store

import (
	"context"
	"encoding/json"
	"html"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

// SGZone is Singapore time: no DST, so a fixed +08:00 offset avoids depending on
// system tzdata (absent in slim containers).
// ponytail: fixed offset, swap to time.LoadLocation if SG ever gets DST.
var SGZone = time.FixedZone("+08", 8*3600)

// Contact is one tender contact, shared across the source scrapers.
type Contact struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

var (
	tagRe = regexp.MustCompile(`<[^>]+>`)
	wsRe  = regexp.MustCompile(`\s+`)
)

// StripHTML removes tags, unescapes entities, and collapses whitespace.
func StripHTML(s string) string {
	s = tagRe.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	return strings.TrimSpace(wsRe.ReplaceAllString(s, " "))
}

// Record is one tender from any source, ready to upsert.
type Record struct {
	Source        string
	SourceID      string
	ReferenceNo   string
	Title         string
	Link          string
	Category      string
	Agency        string
	TenderType    string
	PublishedDate *time.Time
	ClosingDate   *time.Time
	Status        string
	Description   string
	// Details is source-specific structured extra (stored as jsonb). When
	// DetailsFetched is true it's written + details_fetched_at set; otherwise the
	// existing details are preserved (e.g. GeBIZ detail is scraped later by api-server-go).
	Details        any
	DetailsFetched bool
}

const schemaDDL = `
CREATE TABLE IF NOT EXISTS tender (
    "id"                 text PRIMARY KEY,
    "source"             text NOT NULL,
    "source_id"          text NOT NULL,
    "reference_no"       text NOT NULL DEFAULT '',
    "title"              text NOT NULL DEFAULT '',
    "link"               text NOT NULL DEFAULT '',
    "category"           text NOT NULL DEFAULT '',
    "agency"             text NOT NULL DEFAULT '',
    "tender_type"        text NOT NULL DEFAULT '',
    "published_date"     date,
    "closing_date"       timestamptz,
    "status"             text NOT NULL DEFAULT '',
    "description"        text NOT NULL DEFAULT '',
    "details"            jsonb,
    "details_fetched_at" timestamptz,
    "ai_brief"           jsonb,
    "ai_brief_at"        timestamptz,
    "ai_delivery_timeline"    jsonb,
    "ai_delivery_timeline_at" timestamptz,
    "embedding"          vector(384),
    "first_seen_at"      timestamptz NOT NULL DEFAULT now(),
    "last_seen_at"       timestamptz NOT NULL DEFAULT now()
);`

// EnsureSchema creates the tender table (and the vector extension/column/index
// it depends on) if they don't exist yet.
func EnsureSchema(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS vector;`); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, schemaDDL); err != nil {
		return err
	}
	// Older deployments already have the table without the column.
	if _, err := pool.Exec(ctx, `ALTER TABLE tender ADD COLUMN IF NOT EXISTS embedding vector(384);`); err != nil {
		return err
	}
	_, err := pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS tender_embedding_idx ON tender USING hnsw (embedding vector_cosine_ops);`)
	return err
}

// Upsert inserts a tender or refreshes its mutable fields. first_seen_at is never
// overwritten. details/details_fetched_at are only overwritten when the source
// provided them (COALESCE), so GeBIZ's later on-demand scrape isn't clobbered by re-polls.
func Upsert(ctx context.Context, pool *pgxpool.Pool, baseURL, apiKey string, r Record) error {
	id := r.Source + ":" + r.SourceID

	var details, fetchedAt any // nil → SQL NULL
	if r.DetailsFetched {
		b, err := json.Marshal(r.Details)
		if err != nil {
			return err
		}
		details = string(b)
		fetchedAt = time.Now()
	}

	var embedding any // nil → SQL NULL, preserved by COALESCE on re-upsert
	if vec, err := Embed(ctx, baseURL, apiKey, buildEmbedText(r.Title, r.Category, r.Agency, r.Description)); err != nil {
		log.Printf("%s: embed: %v", id, err)
	} else {
		embedding = pgvector.NewVector(vec)
	}

	// Skip inserting a brand-new tender that's already closed (closing_date past);
	// existing rows still upsert via the EXISTS branch → ON CONFLICT.
	// ponytail: embed above still runs for skipped-closed rows (one wasted call);
	// avoiding it would need a pre-existence query — not worth it.
	_, err := pool.Exec(ctx, `
INSERT INTO tender (
    id, source, source_id, reference_no, title, link, category, agency,
    tender_type, published_date, closing_date, status, description,
    details, details_fetched_at, embedding, first_seen_at, last_seen_at
)
SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16, now(), now()
WHERE $11::timestamptz IS NULL OR $11::timestamptz > now()
   OR EXISTS (SELECT 1 FROM tender WHERE id = $1)
ON CONFLICT (id) DO UPDATE SET
    reference_no       = EXCLUDED.reference_no,
    title              = EXCLUDED.title,
    link               = EXCLUDED.link,
    category           = EXCLUDED.category,
    agency             = EXCLUDED.agency,
    tender_type        = EXCLUDED.tender_type,
    published_date     = EXCLUDED.published_date,
    closing_date       = EXCLUDED.closing_date,
    status             = EXCLUDED.status,
    description        = EXCLUDED.description,
    details            = COALESCE(EXCLUDED.details, tender.details),
    details_fetched_at = COALESCE(EXCLUDED.details_fetched_at, tender.details_fetched_at),
    embedding           = COALESCE(EXCLUDED.embedding, tender.embedding),
    last_seen_at       = now();`,
		id, r.Source, r.SourceID, r.ReferenceNo, r.Title, r.Link, r.Category, r.Agency,
		r.TenderType, r.PublishedDate, r.ClosingDate, r.Status, r.Description,
		details, fetchedAt, embedding,
	)
	return err
}
