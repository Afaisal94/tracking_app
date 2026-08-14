package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"tracking-backend/internal/models"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo { return &UserRepo{pool: pool} }

func (r *UserRepo) Create(ctx context.Context, u *models.User) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO users (id, name, email, phone, password_hash, avatar_url)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING created_at, updated_at`,
		u.ID, u.Name, u.Email, u.Phone, u.PasswordHash, u.AvatarURL,
	).Scan(&u.CreatedAt, &u.UpdatedAt)
}

func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	u := &models.User{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, email, phone, password_hash, avatar_url, created_at, updated_at
		FROM users WHERE email=$1`, email).Scan(
		&u.ID, &u.Name, &u.Email, &u.Phone, &u.PasswordHash, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	u := &models.User{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, email, phone, password_hash, avatar_url, created_at, updated_at
		FROM users WHERE id=$1`, id).Scan(
		&u.ID, &u.Name, &u.Email, &u.Phone, &u.PasswordHash, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) Update(ctx context.Context, u *models.User) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE users SET name=$1, phone=$2, avatar_url=$3, updated_at=now()
		WHERE id=$4`, u.Name, u.Phone, u.AvatarURL, u.ID)
	return err
}
