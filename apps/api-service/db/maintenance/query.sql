-- name: GetMaintenanceScope :one
SELECT "enabled", "message", "updatedAt" FROM maintenance_scope WHERE "scope" = $1;

-- name: UpsertMaintenanceScope :one
INSERT INTO maintenance_scope ("scope", "enabled", "message", "updatedAt")
VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
ON CONFLICT ("scope") DO UPDATE SET "enabled" = $2, "message" = $3, "updatedAt" = CURRENT_TIMESTAMP
RETURNING "enabled", "message", "updatedAt";
