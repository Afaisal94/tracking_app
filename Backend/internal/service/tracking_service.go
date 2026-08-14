package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"tracking-backend/internal/models"
	"tracking-backend/internal/repository"
	"tracking-backend/internal/utils"
)

type TrackingService struct {
	repo *repository.TrackingRepo
}

func NewTrackingService(repo *repository.TrackingRepo) *TrackingService {
	return &TrackingService{repo: repo}
}

func (s *TrackingService) StartSession(ctx context.Context, userID uuid.UUID) (*models.TrackingSession, error) {
	// Check if already has active session
	active, err := s.repo.GetActiveSession(ctx, userID)
	if err != nil {
		return nil, err
	}
	if active != nil {
		return active, nil
	}
	now := time.Now()
	session := &models.TrackingSession{
		ID:            uuid.New(),
		UserID:        userID,
		StartedAt:     now,
		Status:        "active",
		TotalDistance: 0,
		SessionDate:   time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC),
	}
	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *TrackingService) EndSession(ctx context.Context, userID uuid.UUID) (*models.TrackingSession, error) {
	active, err := s.repo.GetActiveSession(ctx, userID)
	if err != nil {
		return nil, err
	}
	if active == nil {
		return nil, errors.New("no active session")
	}
	if err := s.repo.EndSession(ctx, active.ID, "ended"); err != nil {
		return nil, err
	}
	_ = s.repo.UpsertDailySummary(ctx, userID, active.SessionDate)
	active.Status = "ended"
	now := time.Now()
	active.EndedAt = &now
	return active, nil
}

func (s *TrackingService) EndSessionByID(ctx context.Context, sessionID uuid.UUID, status string) error {
	return s.repo.EndSession(ctx, sessionID, status)
}

func (s *TrackingService) GetActiveSession(ctx context.Context, userID uuid.UUID) (*models.TrackingSession, error) {
	return s.repo.GetActiveSession(ctx, userID)
}

func (s *TrackingService) AddPoints(ctx context.Context, userID uuid.UUID, req models.BatchPointsRequest) (int, float64, error) {
	if len(req.Points) == 0 {
		return 0, 0, errors.New("no points provided")
	}

	// Resolve session: use provided session_id or active session
	var sessionID uuid.UUID
	if req.Points[0].SessionID != nil && *req.Points[0].SessionID != "" {
		sid, err := uuid.Parse(*req.Points[0].SessionID)
		if err != nil {
			return 0, 0, errors.New("invalid session_id")
		}
		sess, err := s.repo.GetSessionByID(ctx, sid)
		if err != nil {
			return 0, 0, err
		}
		if sess == nil || sess.UserID != userID {
			return 0, 0, errors.New("session not found or not owned")
		}
		sessionID = sid
	} else {
		active, err := s.repo.GetActiveSession(ctx, userID)
		if err != nil {
			return 0, 0, err
		}
		if active == nil {
			// Auto-create session
			active, err = s.StartSession(ctx, userID)
			if err != nil {
				return 0, 0, err
			}
		}
		sessionID = active.ID
	}

	// Get last point for distance calculation
	lastPoint, _ := s.repo.GetLastPoint(ctx, sessionID)

	var points []models.TrackingPoint
	var totalAdded float64
	var prevLat, prevLng float64
	var hasPrev bool
	if lastPoint != nil {
		prevLat = lastPoint.Lat
		prevLng = lastPoint.Lng
		hasPrev = true
	}

	for _, bp := range req.Points {
		if !utils.IsValidLatLng(bp.Lat, bp.Lng) {
			continue
		}
		var dist float64
		if hasPrev {
			dist = utils.Haversine(prevLat, prevLng, bp.Lat, bp.Lng)
			if utils.ShouldIgnorePoint(bp.Accuracy, dist) {
				continue
			}
			totalAdded += dist
		}
		points = append(points, models.TrackingPoint{
			SessionID:  sessionID,
			Lat:        bp.Lat,
			Lng:        bp.Lng,
			Accuracy:   bp.Accuracy,
			Speed:      bp.Speed,
			Altitude:   bp.Altitude,
			RecordedAt: bp.RecordedAt,
		})
		prevLat = bp.Lat
		prevLng = bp.Lng
		hasPrev = true
	}

	if len(points) == 0 {
		return 0, 0, nil
	}

	if err := s.repo.InsertPoints(ctx, points); err != nil {
		return 0, 0, err
	}
	if totalAdded > 0 {
		_ = s.repo.UpdateSessionDistance(ctx, sessionID, totalAdded)
	}

	return len(points), totalAdded, nil
}

func (s *TrackingService) GetHistory(ctx context.Context, userID uuid.UUID, page, perPage int) ([]models.HistoryItem, int, error) {
	if perPage <= 0 || perPage > 100 {
		perPage = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * perPage
	return s.repo.GetHistory(ctx, userID, perPage, offset)
}

func (s *TrackingService) GetDetail(ctx context.Context, userID uuid.UUID, date string) (*models.DetailResponse, error) {
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return nil, errors.New("invalid date format, expected YYYY-MM-DD")
	}
	sessions, err := s.repo.GetSessionsByDate(ctx, userID, date)
	if err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		return nil, errors.New("no data for date")
	}

	var totalDist float64
	var totalDuration int
	var details []models.SessionDetail

	for _, sess := range sessions {
		points, err := s.repo.GetPointsBySession(ctx, sess.ID)
		if err != nil {
			return nil, err
		}
		if points == nil {
			points = []models.TrackingPoint{}
		}
		totalDist += sess.TotalDistance
		end := time.Now()
		if sess.EndedAt != nil {
			end = *sess.EndedAt
		}
		totalDuration += int(end.Sub(sess.StartedAt).Seconds())

		details = append(details, models.SessionDetail{
			SessionID:      sess.ID,
			StartedAt:      sess.StartedAt,
			EndedAt:        sess.EndedAt,
			Status:         sess.Status,
			TotalDistanceM: sess.TotalDistance,
			Points:         points,
		})
	}

	return &models.DetailResponse{
		Date:             date,
		TotalDistanceM:   totalDist,
		TotalDistanceKm:  totalDist / 1000,
		TotalDurationSec: totalDuration,
		Sessions:         details,
	}, nil
}
