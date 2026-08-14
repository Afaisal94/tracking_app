package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"tracking-backend/internal/auth"
	"tracking-backend/internal/config"
	"tracking-backend/internal/models"
	"tracking-backend/internal/repository"
	"tracking-backend/internal/utils"
)

type AuthService struct {
	cfg       *config.Config
	userRepo  *repository.UserRepo
	tokenRepo *repository.TokenRepo
}

func NewAuthService(cfg *config.Config, userRepo *repository.UserRepo, tokenRepo *repository.TokenRepo) *AuthService {
	return &AuthService{cfg: cfg, userRepo: userRepo, tokenRepo: tokenRepo}
}

func (s *AuthService) Register(ctx context.Context, req models.RegisterRequest) (*models.User, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Phone = strings.TrimSpace(req.Phone)

	if req.Name == "" {
		return nil, errors.New("name is required")
	}
	if !utils.IsValidEmail(req.Email) {
		return nil, errors.New("invalid email format")
	}
	if req.Password != req.ConfirmPassword {
		return nil, errors.New("password and confirm password do not match")
	}
	if ok, msg := utils.ValidatePassword(req.Password); !ok {
		return nil, errors.New(msg)
	}

	existing, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("email already registered")
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		ID:           uuid.New(),
		Name:         req.Name,
		Email:        req.Email,
		Phone:        req.Phone,
		PasswordHash: hash,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *AuthService) Login(ctx context.Context, req models.LoginRequest) (*models.AuthResponse, error) {
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if !utils.IsValidEmail(req.Email) {
		return nil, errors.New("invalid email format")
	}
	if req.Password == "" {
		return nil, errors.New("password is required")
	}

	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if user == nil || !auth.CheckPassword(user.PasswordHash, req.Password) {
		return nil, errors.New("invalid credentials")
	}

	return s.generateTokens(ctx, user)
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*models.AuthResponse, error) {
	if refreshToken == "" {
		return nil, errors.New("refresh token is required")
	}
	hash := auth.HashToken(refreshToken)
	t, err := s.tokenRepo.FindByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("invalid refresh token")
		}
		return nil, err
	}
	if t.Revoked {
		return nil, errors.New("refresh token revoked")
	}
	if time.Now().After(t.ExpiresAt) {
		return nil, errors.New("refresh token expired")
	}

	// Rotate: revoke old, create new
	_ = s.tokenRepo.RevokeByHash(ctx, hash)

	user, err := s.userRepo.FindByID(ctx, t.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	return s.generateTokens(ctx, user)
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string, userID uuid.UUID) error {
	if refreshToken != "" {
		hash := auth.HashToken(refreshToken)
		_ = s.tokenRepo.RevokeByHash(ctx, hash)
	}
	// Also revoke all tokens for user if no specific token provided (force logout)
	if refreshToken == "" && userID != uuid.Nil {
		_ = s.tokenRepo.RevokeAllForUser(ctx, userID)
	}
	return nil
}

func (s *AuthService) generateTokens(ctx context.Context, user *models.User) (*models.AuthResponse, error) {
	accessToken, err := auth.GenerateAccessToken(s.cfg, user.ID, user.Email)
	if err != nil {
		return nil, err
	}
	rawRefresh := auth.GenerateRefreshToken()
	hash := auth.HashToken(rawRefresh)

	rt := &models.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(s.cfg.JWTRefreshExpiry),
		Revoked:   false,
	}
	if err := s.tokenRepo.Create(ctx, rt); err != nil {
		return nil, err
	}

	// Return user without password hash
	safeUser := &models.User{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Phone:     user.Phone,
		AvatarURL: user.AvatarURL,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	return &models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresIn:    int64(s.cfg.JWTAccessExpiry.Seconds()),
		User:         safeUser,
	}, nil
}
