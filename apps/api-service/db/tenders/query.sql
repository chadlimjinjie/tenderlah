-- ListTenders/CountTenders (paginated, no filters) were replaced by the
-- hand-written ListTendersSearch/CountTendersSearch in internal/db/tenders_search.go,
-- which need dynamic WHERE/ORDER BY that sqlc can't generate.

-- name: GetTender :one
SELECT * FROM tender WHERE id = $1;

-- name: UpdateTenderDetails :one
UPDATE tender SET details = $2, details_fetched_at = now()
WHERE id = $1 RETURNING *;

-- name: UpdateTenderBrief :one
UPDATE tender SET ai_brief = $2, ai_brief_at = now()
WHERE id = $1 RETURNING *;

-- name: UpdateTenderDeliveryTimeline :one
UPDATE tender SET ai_delivery_timeline = $2, ai_delivery_timeline_at = now()
WHERE id = $1 RETURNING *;

-- name: ListFavouriteTenderIDs :many
SELECT tender_id FROM tender_favourite WHERE user_id = $1;

-- name: CreateTenderFavourite :exec
INSERT INTO tender_favourite (user_id, tender_id) VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: DeleteTenderFavourite :exec
DELETE FROM tender_favourite WHERE user_id = $1 AND tender_id = $2;

-- name: GetTenderMatch :one
SELECT * FROM tender_match WHERE "userId" = $1 AND "tenderId" = $2;

-- name: UpsertTenderMatch :one
INSERT INTO tender_match ("userId", "tenderId", "score", "priority", "fitReasons", "missingRequirements", "risks", "createdAt", "updatedAt")
VALUES ($1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT ("userId", "tenderId") DO UPDATE
SET "score" = $3, "priority" = $4, "fitReasons" = $5, "missingRequirements" = $6, "risks" = $7, "updatedAt" = CURRENT_TIMESTAMP
RETURNING *;
