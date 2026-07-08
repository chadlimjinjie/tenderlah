// tender-worker pulls GeBIZ tender-opportunity RSS feeds and upserts each
// opportunity into Postgres for later analysis. Run once and exit; schedule
// repeats externally (e.g. cron/systemd timer).
package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/midtone/tender-worker/internal/ahtc"
	"github.com/midtone/tender-worker/internal/emservices"
	"github.com/midtone/tender-worker/internal/gebiz"
	"github.com/midtone/tender-worker/internal/mandai"
	"github.com/midtone/tender-worker/internal/mediacorp"
	"github.com/midtone/tender-worker/internal/nationalgallery"
	"github.com/midtone/tender-worker/internal/nus"
	"github.com/midtone/tender-worker/internal/sats"
	"github.com/midtone/tender-worker/internal/singtel"
	"github.com/midtone/tender-worker/internal/sit"
	"github.com/midtone/tender-worker/internal/smrt"
	"github.com/midtone/tender-worker/internal/store"
	"github.com/midtone/tender-worker/internal/tuaspower"
)

// sources are the per-source fetchers; add a new one here to onboard a source.
var sources = map[string]func(context.Context) ([]store.Record, error){
	"gebiz":           gebiz.FetchAll,
	"smrt":            smrt.Fetch,
	"ahtc":            ahtc.Fetch,
	"singtel":         singtel.Fetch,
	"sats":            sats.Fetch,
	"nationalgallery": nationalgallery.Fetch,
	"sit":             sit.Fetch,
	"mediacorp":       mediacorp.Fetch,
	"tuaspower":       tuaspower.Fetch,
	"mandai":          mandai.Fetch,
	"nus":             nus.Fetch,
	"emservices":      emservices.Fetch,
}

func main() {
	_ = godotenv.Load()

	backfillEmbeddings := flag.Bool("backfill-embeddings", false, "re-embed every tender row with NULL embedding and exit")
	flag.Parse()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL not set")
	}
	// Not fatal-if-missing: Upsert degrades to a NULL embedding when the embed
	// call fails, so ingestion still proceeds without it. Points at the
	// internal apps/embed-service (potion-base-8M, 256-dim).
	embedServiceURL := os.Getenv("EMBED_SERVICE_URL")
	if embedServiceURL == "" {
		embedServiceURL = "http://localhost:4107"
	}

	ctx := context.Background()

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("db: parse config: %v", err)
	}
	// Pooler-safe (transaction-mode PgBouncer/Supabase): avoid named prepared
	// statements, same as apps/api-server-go/internal/db/pool.go.
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeCacheDescribe
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	if err := store.EnsureSchema(ctx, pool); err != nil {
		log.Fatalf("db: ensure schema: %v", err)
	}

	if *backfillEmbeddings {
		if err := store.BackfillEmbeddings(ctx, pool, embedServiceURL, ""); err != nil {
			log.Fatalf("backfill: %v", err)
		}
		return
	}

	// ponytail: GeBIZ RSS kill-switch. Ceased by default pending written DSTA permission
	// (see internal/gebiz/COMPLIANCE.md); set GEBIZ_ENABLED=true to resume ingestion.
	if os.Getenv("GEBIZ_ENABLED") != "true" {
		delete(sources, "gebiz")
		log.Print("gebiz: disabled (set GEBIZ_ENABLED=true to resume)")
	}

	log.Printf("tender-worker: %d source(s)", len(sources))
	pollAll(ctx, pool, embedServiceURL)
}

func pollAll(ctx context.Context, pool *pgxpool.Pool, embedServiceURL string) {
	for name, fetch := range sources {
		records, err := fetch(ctx)
		if err != nil {
			log.Printf("%s: fetch: %v", name, err)
			continue
		}
		var n int
		for _, r := range records {
			if err := store.Upsert(ctx, pool, embedServiceURL, "", r); err != nil {
				log.Printf("%s: upsert %s: %v", name, r.SourceID, err)
				continue
			}
			n++
		}
		log.Printf("%s: upserted %d/%d", name, n, len(records))
	}
}
