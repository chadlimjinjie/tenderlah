-- name: GetUserOnboarding :one
SELECT * FROM user_onboarding WHERE "userId" = $1;

-- name: UpsertUserOnboarding :one
INSERT INTO user_onboarding ("userId", "accountType", "orgName", "businessName", "businessDescription", "industry", "companySize", "geographicCoverage", "preferredContractSize", "createdAt", "updatedAt")
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT ("userId") DO UPDATE
SET "accountType" = $2, "orgName" = $3, "businessName" = $4, "businessDescription" = $5, "industry" = $6,
    "companySize" = $7, "geographicCoverage" = $8, "preferredContractSize" = $9, "updatedAt" = CURRENT_TIMESTAMP
RETURNING *;

-- name: SetUserOnboardingLogo :one
UPDATE user_onboarding SET "logoUrl" = $2, "updatedAt" = CURRENT_TIMESTAMP
WHERE "userId" = $1
RETURNING *;
