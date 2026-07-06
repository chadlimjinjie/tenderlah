package db

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pgvector/pgvector-go"
)

// ponytail: GeBIZ is hidden from the public listing pending written DSTA permission
// (see apps/tender-worker/internal/gebiz/COMPLIANCE.md). Flip GEBIZ_LISTING_ENABLED=true
// to show it again — no code change needed.
func gebizListingEnabled() bool { return os.Getenv("GEBIZ_LISTING_ENABLED") == "true" }

// Hand-written (not sqlc) because the WHERE/ORDER BY are conditional on
// optional filters; sortColumn/sortDir are restricted to fixed whitelists
// below so they can be concatenated into the query safely.
type ListTendersSearchParams struct {
	Q                  string // matched against title/agency/reference_no via ILIKE
	Status             string
	OpenState          string // "open" | "closed" (by closing_date vs now); "" = all
	Source             string
	Agency             string
	SortBy             string // "closingDate" | "publishedDate" | "title" | "relevance"
	SortDir            string // "asc" | "desc"
	FavouritesOnly     bool
	UserID             string           // required when FavouritesOnly is set
	RelevanceEmbedding *pgvector.Vector // signed-in user's onboarding-profile embedding, used when SortBy is "relevance"
	Limit              int32
	Offset             int32
}

var tenderSortColumns = map[string]string{
	"closingDate":   "closing_date",
	"publishedDate": "published_date",
	"title":         "title",
}

// tenderSortClause builds the ORDER BY clause, appending any parameters it needs
// (e.g. the relevance embedding) to args so the caller can size LIMIT/OFFSET
// placeholders correctly afterwards.
//
// relevance ranks by cosine distance (same operator/shape as
// ListRecommendedTenders' favourite-centroid ordering) between each tender's
// embedding and the caller-supplied profile embedding. Falls back to recency
// when there's no embedding to compare against (anonymous visitors, or users
// who haven't onboarded / have an empty profile).
func tenderSortClause(p ListTendersSearchParams, args *[]any) string {
	if p.SortBy == "relevance" {
		if p.RelevanceEmbedding != nil {
			*args = append(*args, *p.RelevanceEmbedding)
			n := len(*args)
			return fmt.Sprintf(
				"(CASE WHEN embedding IS NOT NULL THEN embedding <=> $%d END) ASC NULLS LAST, published_date DESC NULLS LAST, id DESC",
				n)
		}
		return "published_date DESC NULLS LAST, id DESC"
	}

	col, ok := tenderSortColumns[p.SortBy]
	if !ok {
		col = "published_date"
	}
	dir := "DESC"
	if p.SortDir == "asc" {
		dir = "ASC"
	}
	// Secondary relevance order: the chosen column stays primary, but signed-in
	// users still get tenders ranked against their profile within each column value.
	relevance := ""
	if p.RelevanceEmbedding != nil {
		*args = append(*args, *p.RelevanceEmbedding)
		relevance = fmt.Sprintf(", (CASE WHEN embedding IS NOT NULL THEN embedding <=> $%d END) ASC NULLS LAST", len(*args))
	}
	// id tie-breaker: published_date/closing_date/title aren't unique, so without
	// it OFFSET paging can repeat or skip rows whenever ties reorder between queries.
	return fmt.Sprintf("%s %s NULLS LAST%s, id %s", col, dir, relevance, dir)
}

// splitMulti parses a pipe-separated multi-select filter value into its non-empty,
// trimmed parts (delimiter must match the frontend's FilterPicker join).
func splitMulti(s string) []string {
	var out []string
	for _, v := range strings.Split(s, "|") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func tenderSearchWhere(p ListTendersSearchParams, args *[]any) string {
	var clauses []string
	if !gebizListingEnabled() {
		clauses = append(clauses, "source <> 'gebiz'")
	}
	if p.Q != "" {
		*args = append(*args, "%"+p.Q+"%")
		n := len(*args)
		clauses = append(clauses, fmt.Sprintf("(title ILIKE $%d OR agency ILIKE $%d OR reference_no ILIKE $%d)", n, n, n))
	}
	if p.Status != "" {
		*args = append(*args, p.Status)
		clauses = append(clauses, fmt.Sprintf("status = $%d", len(*args)))
	}
	// open/closed is by closing deadline, not the inconsistent source status text:
	// closed = past its closing date; open = upcoming or no deadline set.
	switch p.OpenState {
	case "open":
		clauses = append(clauses, "(closing_date IS NULL OR closing_date >= now())")
	case "closed":
		clauses = append(clauses, "closing_date < now()")
	}
	// source accepts a pipe-separated list (e.g. "ahtc|mandai|nus") → match any.
	// Pipe, not comma: agency names can contain commas (see agency filter below).
	if vals := splitMulti(p.Source); len(vals) > 0 {
		*args = append(*args, vals)
		clauses = append(clauses, fmt.Sprintf("source = ANY($%d)", len(*args)))
	}
	// agency accepts a pipe-separated list → match any (names can contain commas).
	if vals := splitMulti(p.Agency); len(vals) > 0 {
		*args = append(*args, vals)
		clauses = append(clauses, fmt.Sprintf("agency = ANY($%d)", len(*args)))
	}
	if p.FavouritesOnly {
		*args = append(*args, p.UserID)
		clauses = append(clauses, fmt.Sprintf("EXISTS (SELECT 1 FROM tender_favourite tf WHERE tf.tender_id = tender.id AND tf.user_id = $%d)", len(*args)))
	}
	if len(clauses) == 0 {
		return ""
	}
	return "WHERE " + strings.Join(clauses, " AND ")
}

func (q *Queries) ListTendersSearch(ctx context.Context, p ListTendersSearchParams) ([]Tender, error) {
	var args []any
	where := tenderSearchWhere(p, &args)
	orderBy := tenderSortClause(p, &args)
	args = append(args, p.Limit, p.Offset)
	query := fmt.Sprintf(
		`SELECT id, source, source_id, reference_no, title, link, category, agency, tender_type,
		        published_date, closing_date, status, description, first_seen_at, last_seen_at
		 FROM tender %s ORDER BY %s LIMIT $%d OFFSET $%d`,
		where, orderBy, len(args)-1, len(args))

	rows, err := q.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Tender
	for rows.Next() {
		var i Tender
		if err := rows.Scan(&i.ID, &i.Source, &i.SourceID, &i.ReferenceNo, &i.Title, &i.Link, &i.Category, &i.Agency, &i.TenderType, &i.PublishedDate, &i.ClosingDate, &i.Status, &i.Description, &i.FirstSeenAt, &i.LastSeenAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

func (q *Queries) CountTendersSearch(ctx context.Context, p ListTendersSearchParams) (int64, error) {
	var args []any
	where := tenderSearchWhere(p, &args)
	query := fmt.Sprintf(`SELECT count(*) FROM tender %s`, where)
	row := q.db.QueryRow(ctx, query, args...)
	var count int64
	err := row.Scan(&count)
	return count, err
}

func (q *Queries) ListDistinctTenderStatuses(ctx context.Context) ([]string, error) {
	return tenderDistinctColumn(ctx, q, "status")
}

func (q *Queries) ListDistinctTenderSources(ctx context.Context) ([]string, error) {
	query := `SELECT DISTINCT source FROM tender WHERE source != ''`
	if !gebizListingEnabled() {
		query += " AND source <> 'gebiz'"
	}
	query += " ORDER BY source"
	return tenderDistinctQuery(ctx, q, query)
}

// source is the same pipe-separated multi-value convention as the search filters
// (see tenderSearchWhere); when non-empty, agencies are scoped to those sources so
// the frontend's Agency dropdown only lists agencies under the selected Source(s).
func (q *Queries) ListDistinctTenderAgencies(ctx context.Context, source string) ([]string, error) {
	query := `SELECT DISTINCT agency FROM tender WHERE agency != ''`
	if !gebizListingEnabled() {
		query += " AND source <> 'gebiz'"
	}
	var args []any
	if vals := splitMulti(source); len(vals) > 0 {
		args = append(args, vals)
		query += " AND source = ANY($1)"
	}
	query += " ORDER BY agency"

	rows, err := q.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var values []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	return values, rows.Err()
}

// col is always one of the two literals above, never user input.
func tenderDistinctColumn(ctx context.Context, q *Queries, col string) ([]string, error) {
	return tenderDistinctQuery(ctx, q, fmt.Sprintf(`SELECT DISTINCT %s FROM tender WHERE %s != '' ORDER BY %s`, col, col, col))
}

// tenderDistinctQuery runs a parameterless single-string-column query and collects
// the results. Query text is caller-built from literals only, never user input.
func tenderDistinctQuery(ctx context.Context, q *Queries, query string) ([]string, error) {
	rows, err := q.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var values []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	return values, rows.Err()
}
