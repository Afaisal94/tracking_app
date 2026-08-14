package middleware

import (
	"context"
	"net/http"
	"strings"

	"tracking-backend/internal/auth"
	"tracking-backend/internal/config"
	"tracking-backend/internal/utils"

	"github.com/google/uuid"
)

type contextKey string

const UserIDKey contextKey = "user_id"
const UserEmailKey contextKey = "user_email"

func AuthMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				utils.Error(w, http.StatusUnauthorized, "missing authorization header")
				return
			}
			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				utils.Error(w, http.StatusUnauthorized, "invalid authorization header format")
				return
			}
			tokenStr := strings.TrimSpace(parts[1])
			if tokenStr == "" {
				utils.Error(w, http.StatusUnauthorized, "missing token")
				return
			}
			claims, err := auth.ParseAccessToken(cfg, tokenStr)
			if err != nil {
				utils.Error(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, UserEmailKey, claims.Email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(r *http.Request) (uuid.UUID, bool) {
	v := r.Context().Value(UserIDKey)
	if v == nil {
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok
}
