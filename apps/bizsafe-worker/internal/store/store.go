// Package store owns the bizsafe_companies table and its upsert.
package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Record is one bizSAFE certificate row, ready to upsert.
type Record struct {
	Cert        string
	UEN         string
	CompanyName string
	Type        string
	Active      string
	Expiry      *time.Time
	Status      string
	Approval    *time.Time
	Level       string
	Description string
}

const schemaDDL = `
CREATE TABLE IF NOT EXISTS bizsafe_companies (
    "cert"          text PRIMARY KEY,
    "uen"           text NOT NULL DEFAULT '',
    "company_name"  text NOT NULL DEFAULT '',
    "type"          text NOT NULL DEFAULT '',
    "active"        text NOT NULL DEFAULT '',
    "expiry"        date,
    "status"        text NOT NULL DEFAULT '',
    "approval"      date,
    "level"         text NOT NULL DEFAULT '',
    "description"   text NOT NULL DEFAULT '',
    "first_seen_at" timestamptz NOT NULL DEFAULT now(),
    "last_seen_at"  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS bizsafe_companies_uen_idx ON bizsafe_companies (uen);`

// EnsureSchema creates the bizsafe_companies table and its index if they don't exist yet.
func EnsureSchema(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, schemaDDL)
	return err
}

// Upsert inserts a certificate row or refreshes its mutable fields. first_seen_at is
// never overwritten; last_seen_at advances on every successful re-import.
func Upsert(ctx context.Context, pool *pgxpool.Pool, r Record) error {
	_, err := pool.Exec(ctx, `
INSERT INTO bizsafe_companies (
    cert, uen, company_name, type, active, expiry, status, approval, level, description,
    first_seen_at, last_seen_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10, now(), now())
ON CONFLICT (cert) DO UPDATE SET
    uen          = EXCLUDED.uen,
    company_name = EXCLUDED.company_name,
    type         = EXCLUDED.type,
    active       = EXCLUDED.active,
    expiry       = EXCLUDED.expiry,
    status       = EXCLUDED.status,
    approval     = EXCLUDED.approval,
    level        = EXCLUDED.level,
    description  = EXCLUDED.description,
    last_seen_at = now();`,
		r.Cert, r.UEN, r.CompanyName, r.Type, r.Active, r.Expiry, r.Status, r.Approval, r.Level, r.Description,
	)
	return err
}
