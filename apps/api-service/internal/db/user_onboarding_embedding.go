package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/pgvector/pgvector-go"
)

// Hand-written (not sqlc) because sqlc drops `vector` columns from generated
// models — same reason tenders_search.go is hand-written. The stored vector is
// an all-minilm embedding of the user's onboarding profile, kept in the same
// space as tender.embedding so relevance ranking can cosine-compare the two.

// SetUserOnboardingEmbedding stores the profile embedding for a user.
func (q *Queries) SetUserOnboardingEmbedding(ctx context.Context, userID string, vec pgvector.Vector) error {
	_, err := q.db.Exec(ctx, `UPDATE user_onboarding SET "embedding" = $2 WHERE "userId" = $1`, userID, vec)
	return err
}

// GetUserOnboardingEmbedding returns the stored profile embedding, or nil when
// the user has no onboarding row or the column is still NULL (onboarded before
// embeddings were precomputed). Read as text (the vector type isn't registered
// with pgx) and parsed via Vector.Scan — the universally-handled path.
func (q *Queries) GetUserOnboardingEmbedding(ctx context.Context, userID string) (*pgvector.Vector, error) {
	var s *string
	err := q.db.QueryRow(ctx, `SELECT "embedding"::text FROM user_onboarding WHERE "userId" = $1`, userID).Scan(&s)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, nil
	}
	var vec pgvector.Vector
	if err := vec.Scan(*s); err != nil {
		return nil, err
	}
	return &vec, nil
}
