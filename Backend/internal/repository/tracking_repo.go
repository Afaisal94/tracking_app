package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"tracking-backend/internal/models"
)

type TrackingRepo struct {
	pool *pgxpool.Pool
}

func NewTrackingRepo(pool *pgxpool.Pool) *TrackingRepo { return &TrackingRepo{pool: pool} }

// Sessions

func (r *TrackingRepo) CreateSession(ctx context.Context, s *models.TrackingSession) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO tracking_sessions (id, user_id, started_at, status, total_distance_m, session_date)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING created_at`, s.ID, s.UserID, s.StartedAt, s.Status, s.TotalDistance, s.SessionDate,
	).Scan(&s.CreatedAt)
}

func (r *TrackingRepo) GetActiveSession(ctx context.Context, userID uuid.UUID) (*models.TrackingSession, error) {
	s := &models.TrackingSession{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, started_at, ended_at, status, total_distance_m, session_date, created_at
		FROM tracking_sessions WHERE user_id=$1 AND status='active' ORDER BY started_at DESC LIMIT 1`, userID,
	).Scan(&s.ID, &s.UserID, &s.StartedAt, &s.EndedAt, &s.Status, &s.TotalDistance, &s.SessionDate, &s.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return s, nil
}

func (r *TrackingRepo) GetSessionByID(ctx context.Context, id uuid.UUID) (*models.TrackingSession, error) {
	s := &models.TrackingSession{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, started_at, ended_at, status, total_distance_m, session_date, created_at
		FROM tracking_sessions WHERE id=$1`, id,
	).Scan(&s.ID, &s.UserID, &s.StartedAt, &s.EndedAt, &s.Status, &s.TotalDistance, &s.SessionDate, &s.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return s, nil
}

func (r *TrackingRepo) EndSession(ctx context.Context, sessionID uuid.UUID, status string) error {
	now := time.Now()
	_, err := r.pool.Exec(ctx, `
		UPDATE tracking_sessions SET status=$1, ended_at=$2 WHERE id=$3`, status, now, sessionID)
	return err
}

func (r *TrackingRepo) UpdateSessionDistance(ctx context.Context, sessionID uuid.UUID, distance float64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE tracking_sessions SET total_distance_m = total_distance_m + $1 WHERE id=$2`, distance, sessionID)
	return err
}

func (r *TrackingRepo) SetSessionDistance(ctx context.Context, sessionID uuid.UUID, distance float64) error {
	_, err := r.pool.Exec(ctx, `UPDATE tracking_sessions SET total_distance_m=$1 WHERE id=$2`, distance, sessionID)
	return err
}

// Points

func (r *TrackingRepo) InsertPoints(ctx context.Context, points []models.TrackingPoint) error {
	if len(points) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, p := range points {
		batch.Queue(`
			INSERT INTO tracking_points (session_id, lat, lng, accuracy, speed, altitude, recorded_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			p.SessionID, p.Lat, p.Lng, p.Accuracy, p.Speed, p.Altitude, p.RecordedAt)
	}
	results := r.pool.SendBatch(ctx, batch)
	defer results.Close()
	for range points {
		if _, err := results.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func (r *TrackingRepo) GetPointsBySession(ctx context.Context, sessionID uuid.UUID) ([]models.TrackingPoint, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, session_id, lat, lng, accuracy, speed, altitude, recorded_at, created_at
		FROM tracking_points WHERE session_id=$1 ORDER BY recorded_at ASC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var points []models.TrackingPoint
	for rows.Next() {
		var p models.TrackingPoint
		if err := rows.Scan(&p.ID, &p.SessionID, &p.Lat, &p.Lng, &p.Accuracy, &p.Speed, &p.Altitude, &p.RecordedAt, &p.CreatedAt); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

func (r *TrackingRepo) GetLastPoint(ctx context.Context, sessionID uuid.UUID) (*models.TrackingPoint, error) {
	p := &models.TrackingPoint{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, session_id, lat, lng, accuracy, speed, altitude, recorded_at, created_at
		FROM tracking_points WHERE session_id=$1 ORDER BY recorded_at DESC LIMIT 1`, sessionID,
	).Scan(&p.ID, &p.SessionID, &p.Lat, &p.Lng, &p.Accuracy, &p.Speed, &p.Altitude, &p.RecordedAt, &p.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

// History

func (r *TrackingRepo) GetHistory(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.HistoryItem, int, error) {
	// Total count of distinct dates
	var total int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM (
			SELECT session_date FROM tracking_sessions WHERE user_id=$1 GROUP BY session_date
		) t`, userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT session_date,
		       COALESCE(SUM(total_distance_m),0) as total_distance,
		       COALESCE(SUM(EXTRACT(EPOCH FROM (COALESCE(ended_at, now()) - started_at))::int),0) as total_duration,
		       COUNT(*) as session_count
		FROM tracking_sessions
		WHERE user_id=$1
		GROUP BY session_date
		ORDER BY session_date DESC
		LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []models.HistoryItem
	for rows.Next() {
		var d time.Time
		var dist float64
		var dur int
		var cnt int
		if err := rows.Scan(&d, &dist, &dur, &cnt); err != nil {
			return nil, 0, err
		}
		items = append(items, models.HistoryItem{
			Date:             d.Format("2006-01-02"),
			TotalDistanceM:   dist,
			TotalDistanceKm:  dist / 1000,
			TotalDurationSec: dur,
			SessionCount:     cnt,
		})
	}
	if items == nil {
		items = []models.HistoryItem{}
	}
	return items, total, rows.Err()
}

func (r *TrackingRepo) GetSessionsByDate(ctx context.Context, userID uuid.UUID, date string) ([]models.TrackingSession, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, started_at, ended_at, status, total_distance_m, session_date, created_at
		FROM tracking_sessions WHERE user_id=$1 AND session_date=$2::date ORDER BY started_at ASC`, userID, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sessions []models.TrackingSession
	for rows.Next() {
		var s models.TrackingSession
		if err := rows.Scan(&s.ID, &s.UserID, &s.StartedAt, &s.EndedAt, &s.Status, &s.TotalDistance, &s.SessionDate, &s.CreatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

// Stale sessions for cleanup
func (r *TrackingRepo) GetStaleActiveSessions(ctx context.Context, before time.Time) ([]models.TrackingSession, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, started_at, ended_at, status, total_distance_m, session_date, created_at
		FROM tracking_sessions WHERE status='active' AND started_at < $1`, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sessions []models.TrackingSession
	for rows.Next() {
		var s models.TrackingSession
		if err := rows.Scan(&s.ID, &s.UserID, &s.StartedAt, &s.EndedAt, &s.Status, &s.TotalDistance, &s.SessionDate, &s.CreatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

// Daily summary upsert
func (r *TrackingRepo) UpsertDailySummary(ctx context.Context, userID uuid.UUID, date time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO daily_mileage_summary (user_id, summary_date, total_distance_m, total_duration_sec, session_count)
		SELECT user_id, session_date,
		       COALESCE(SUM(total_distance_m),0),
		       COALESCE(SUM(EXTRACT(EPOCH FROM (COALESCE(ended_at, now()) - started_at))::int),0),
		       COUNT(*)
		FROM tracking_sessions WHERE user_id=$1 AND session_date=$2::date GROUP BY user_id, session_date
		ON CONFLICT (user_id, summary_date) DO UPDATE SET
			total_distance_m = EXCLUDED.total_distance_m,
			total_duration_sec = EXCLUDED.total_duration_sec,
			session_count = EXCLUDED.session_count
	`, userID, date.Format("2006-01-02"))
	return err
}
