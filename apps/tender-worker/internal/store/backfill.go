package store

import (
	"context"
	"log"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

// buildEmbedText composes the weighted text passed to Embed. category/agency
// are repeated to outweigh the often-long, boilerplate-heavy description so
// the category signal isn't drowned out. Shared by Upsert and
// BackfillEmbeddings so the formula can't drift between the two.
func buildEmbedText(title, category, agency, description string) string {
	return title + " " + strings.Repeat(category+" ", 3) + strings.Repeat(agency+" ", 3) + description
}

// BackfillEmbeddings re-embeds every tender row with a NULL embedding via
// Ollama. Intended to run once after migrate_change_embedding_dim.sql (which
// NULLs all existing embeddings), and safe to re-run any time (e.g. after an
// Ollama outage left some rows NULL from Upsert) since it only touches NULL
// rows. Sequential, not batched/concurrent — low thousands of rows, one-off
// maintenance, not latency-sensitive.
func BackfillEmbeddings(ctx context.Context, pool *pgxpool.Pool, baseURL, apiKey string) error {
	rows, err := pool.Query(ctx, `SELECT id, title, category, agency, description FROM tender WHERE embedding IS NULL ORDER BY id`)
	if err != nil {
		return err
	}
	type pending struct {
		id, title, category, agency, description string
	}
	var items []pending
	for rows.Next() {
		var p pending
		if err := rows.Scan(&p.id, &p.title, &p.category, &p.agency, &p.description); err != nil {
			rows.Close()
			return err
		}
		items = append(items, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	var done int
	for _, p := range items {
		vec, err := Embed(ctx, baseURL, apiKey, buildEmbedText(p.title, p.category, p.agency, p.description))
		if err != nil {
			log.Printf("backfill: %s: embed: %v", p.id, err)
			continue
		}
		if _, err := pool.Exec(ctx, `UPDATE tender SET embedding = $1 WHERE id = $2`, pgvector.NewVector(vec), p.id); err != nil {
			log.Printf("backfill: %s: update: %v", p.id, err)
			continue
		}
		done++
	}
	log.Printf("backfill: %d/%d rows embedded", done, len(items))
	return nil
}
