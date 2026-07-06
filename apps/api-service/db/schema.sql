CREATE EXTENSION IF NOT EXISTS vector;

-- Per-app maintenance flags keyed by a free-form "scope" — e.g. "beta" for the
-- web-beta frontend, "portfolio" for the web-chad portfolio. When a scope is
-- enabled, that frontend shows a full-screen maintenance page; "message" is the
-- optional custom text. Each app reads/writes its own scope, so toggling one
-- never affects another.
CREATE TABLE IF NOT EXISTS maintenance_scope (
    "scope"     text PRIMARY KEY,
    "enabled"   boolean NOT NULL DEFAULT false,
    "message"   text NOT NULL DEFAULT '',
    "updatedAt" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- One row per auth-service user (no FK — different service/DB), collected by
-- the post-signup onboarding flow in apps/web-tenders.
CREATE TABLE IF NOT EXISTS user_onboarding (
    "userId"       text PRIMARY KEY,
    "accountType"  text NOT NULL,
    "orgName"      text NOT NULL,
    "businessName" text,
    "businessDescription" text,
    "industry"     text NOT NULL,
    "logoUrl"      text,
    "companySize"           text, -- e.g. "1-10","11-50","51-200","201-500","500+"
    "geographicCoverage"    text, -- free text, e.g. "Singapore-wide", "Jurong East"
    "preferredContractSize" text, -- e.g. "<50K","50K-200K","200K-1M",">1M"
    -- Precomputed all-minilm embedding of (industry×3 + businessDescription),
    -- written on profile save. Same vector space as tender.embedding so tender
    -- relevance ranking can cosine-compare without a per-request embed call.
    "embedding"    vector(384),
    "createdAt"    timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt"    timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- No migration runner in this repo; schema.sql is applied manually, so
-- existing databases need this explicit ALTER alongside the CREATE above.
ALTER TABLE user_onboarding
    ADD COLUMN IF NOT EXISTS "companySize" text,
    ADD COLUMN IF NOT EXISTS "geographicCoverage" text,
    ADD COLUMN IF NOT EXISTS "preferredContractSize" text,
    ADD COLUMN IF NOT EXISTS "embedding" vector(384);

-- Owned/written by apps/tender-worker (EnsureSchema). Declared here for SQLC
-- codegen only — api-server-go reads it but never creates/migrates it. Unified
-- across sources (gebiz, smrt, …); id = source || ':' || source_id.
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
    "embedding"          vector(128),
    "first_seen_at"      timestamptz NOT NULL DEFAULT now(),
    "last_seen_at"       timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS tender_favourite (
    "user_id"    text NOT NULL,
    "tender_id"  text NOT NULL REFERENCES tender(id),
    "created_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("user_id", "tender_id")
);

-- Per-(user, tender) AI match score; owned by api-server-go. Distinct from
-- tender.ai_brief (supplier-agnostic, cached once per tender) because fit
-- depends on the viewer's company profile.
CREATE TABLE IF NOT EXISTS tender_match (
    "userId"               text NOT NULL,
    "tenderId"             text NOT NULL REFERENCES tender(id),
    "score"                integer NOT NULL,
    "priority"             text NOT NULL, -- HIGH | MEDIUM | LOW
    "fitReasons"           jsonb NOT NULL,
    "missingRequirements"  jsonb NOT NULL,
    "risks"                jsonb NOT NULL,
    "createdAt"            timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt"            timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("userId", "tenderId")
);
