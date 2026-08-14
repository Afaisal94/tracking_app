package models

import (
	"time"

	"github.com/google/uuid"
)

// User
type User struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone,omitempty"`
	PasswordHash string    `json:"-"`
	AvatarURL    *string   `json:"avatar_url,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// RefreshToken
type RefreshToken struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	TokenHash string    `json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	Revoked   bool      `json:"revoked"`
	CreatedAt time.Time `json:"created_at"`
}

// TrackingSession
type TrackingSession struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"user_id"`
	StartedAt     time.Time  `json:"started_at"`
	EndedAt       *time.Time `json:"ended_at,omitempty"`
	Status        string     `json:"status"` // active | ended | disconnected
	TotalDistance float64    `json:"total_distance_m"`
	SessionDate   time.Time  `json:"session_date"`
	CreatedAt     time.Time  `json:"created_at"`
}

// TrackingPoint
type TrackingPoint struct {
	ID         int64     `json:"id"`
	SessionID  uuid.UUID `json:"session_id"`
	Lat        float64   `json:"lat"`
	Lng        float64   `json:"lng"`
	Accuracy   *float64  `json:"accuracy,omitempty"`
	Speed      *float64  `json:"speed,omitempty"`
	Altitude   *float64  `json:"altitude,omitempty"`
	RecordedAt time.Time `json:"recorded_at"`
	CreatedAt  time.Time `json:"created_at"`
}

// DailyMileageSummary
type DailyMileageSummary struct {
	UserID           uuid.UUID `json:"user_id"`
	SummaryDate      time.Time `json:"summary_date"`
	TotalDistanceM   float64   `json:"total_distance_m"`
	TotalDurationSec int       `json:"total_duration_sec"`
	SessionCount     int       `json:"session_count"`
}

// DTOs

type RegisterRequest struct {
	Name            string `json:"name"`
	Email           string `json:"email"`
	Phone           string `json:"phone"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	User         *User  `json:"user,omitempty"`
}

type UpdateProfileRequest struct {
	Name      *string `json:"name"`
	Phone     *string `json:"phone"`
	AvatarURL *string `json:"avatar_url"`
}

type HistoryItem struct {
	Date             string  `json:"date"` // YYYY-MM-DD
	TotalDistanceM   float64 `json:"total_distance_m"`
	TotalDistanceKm  float64 `json:"total_distance_km"`
	TotalDurationSec int     `json:"total_duration_sec"`
	SessionCount     int     `json:"session_count"`
}

type DetailResponse struct {
	Date             string           `json:"date"`
	TotalDistanceM   float64          `json:"total_distance_m"`
	TotalDistanceKm  float64          `json:"total_distance_km"`
	TotalDurationSec int              `json:"total_duration_sec"`
	Sessions         []SessionDetail  `json:"sessions"`
}

type SessionDetail struct {
	SessionID      uuid.UUID       `json:"session_id"`
	StartedAt      time.Time       `json:"started_at"`
	EndedAt        *time.Time      `json:"ended_at,omitempty"`
	Status         string          `json:"status"`
	TotalDistanceM float64         `json:"total_distance_m"`
	Points         []TrackingPoint `json:"points"`
}

type BatchPointsRequest struct {
	Points []BatchPoint `json:"points"`
}

type BatchPoint struct {
	SessionID  *string  `json:"session_id,omitempty"`
	Lat        float64  `json:"lat"`
	Lng        float64  `json:"lng"`
	Accuracy   *float64 `json:"accuracy,omitempty"`
	Speed      *float64 `json:"speed,omitempty"`
	Altitude   *float64 `json:"altitude,omitempty"`
	RecordedAt time.Time `json:"recorded_at"`
}

// WebSocket MessagePack structs

type WSMessage struct {
	Type      string   `msgpack:"type" json:"type"`
	SessionID string   `msgpack:"session_id,omitempty" json:"session_id,omitempty"`
	Lat       *float64 `msgpack:"lat,omitempty" json:"lat,omitempty"`
	Lng       *float64 `msgpack:"lng,omitempty" json:"lng,omitempty"`
	Accuracy  *float64 `msgpack:"accuracy,omitempty" json:"accuracy,omitempty"`
	Speed     *float64 `msgpack:"speed,omitempty" json:"speed,omitempty"`
	Altitude  *float64 `msgpack:"altitude,omitempty" json:"altitude,omitempty"`
	Timestamp *string  `msgpack:"timestamp,omitempty" json:"timestamp,omitempty"`
	V         *int     `msgpack:"v,omitempty" json:"v,omitempty"`
}

type WSAck struct {
	Type      string `msgpack:"type" json:"type"`
	Status    string `msgpack:"status" json:"status"`
	SessionID string `msgpack:"session_id,omitempty" json:"session_id,omitempty"`
	Message   string `msgpack:"message,omitempty" json:"message,omitempty"`
}
