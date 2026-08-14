package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"tracking-backend/internal/models"
)

type TokenRepo struct {
	pool *pgxpool.Pool
}

func NewTokenRepo(pool *pgxpool.Pool) *TokenRepo { return &TokenRepo{pool: pool} }

func (r *TokenRepo) Create(ctx context.Context, t *models.RefreshToken) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, revoked)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING created_at`, t.ID, t.UserID, t.TokenHash, t.ExpiresAt, t.Revoked,
	).Scan(&t.CreatedAt)
}

func (r *TokenRepo) FindByHash(ctx context.Context, hash string) (*models.RefreshToken, error) {
	t := &models.RefreshToken{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, token_hash, expires_at, revoked, created_at
		FROM refresh_tokens WHERE token_hash=$1`, hash).Scan(
		&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.Revoked, &t.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *TokenRepo) RevokeByHash(ctx context.Context, hash string) error {
	_, err := r.pool.Exec(ctx, `UPDATE refresh_tokens SET revoked=true WHERE token_hash=$1`, hash)
	return err
}

func (r *TokenRepo) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE refresh_tokens SET revoked=true WHERE user_id=$1 AND revoked=false`, userID)
	return err
}

func (r *TokenRepo) DeleteExpired(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE expires_at < $1 OR revoked=true`, time.Now())
	return err
}
