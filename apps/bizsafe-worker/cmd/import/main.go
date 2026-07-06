// bizsafe-worker is a one-shot importer: it scrapes the WSH Council's
// bizSAFE Self-Help Excel file and upserts every certificate row into
// Postgres (bizsafe_companies table). Run on demand, not scheduled.
package main

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/midtone/bizsafe-worker/internal/bizsafe"
	"github.com/midtone/bizsafe-worker/internal/store"
)

func main() {
	_ = godotenv.Load()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL not set")
	}

	ctx := context.Background()

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("db: parse config: %v", err)
	}
	// Pooler-safe (transaction-mode PgBouncer/Supabase): avoid named prepared
	// statements, same as apps/tender-worker.
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeCacheDescribe
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	if err := store.EnsureSchema(ctx, pool); err != nil {
		log.Fatalf("db: ensure schema: %v", err)
	}

	log.Println("bizsafe-worker: fetching bizSAFE Self-Help file")
	records, err := bizsafe.Fetch(ctx)
	if err != nil {
		log.Fatalf("fetch: %v", err)
	}
	log.Printf("bizsafe-worker: parsed %d rows, upserting", len(records))

	var n, skipped int
	for _, r := range records {
		if !strings.EqualFold(r.Status, "Approved") {
			skipped++
			continue
		}
		if err := store.Upsert(ctx, pool, r); err != nil {
			log.Printf("upsert %s: %v", r.Cert, err)
			continue
		}
		n++
	}
	log.Printf("bizsafe-worker: upserted %d/%d (skipped %d non-approved)", n, len(records), skipped)
}
