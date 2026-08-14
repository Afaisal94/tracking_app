package worker

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"tracking-backend/internal/repository"
	"tracking-backend/internal/service"
)

type CleanupWorker struct {
	redisSvc     *service.RedisService
	trackingRepo *repository.TrackingRepo
	idleTimeout  time.Duration
}

func NewCleanupWorker(redisSvc *service.RedisService, trackingRepo *repository.TrackingRepo, idleTimeout time.Duration) *CleanupWorker {
	return &CleanupWorker{redisSvc: redisSvc, trackingRepo: trackingRepo, idleTimeout: idleTimeout}
}

func (w *CleanupWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	log.Info().Dur("idle_timeout", w.idleTimeout).Msg("cleanup worker started")
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("cleanup worker stopped")
			return
		case <-ticker.C:
			w.runOnce(ctx)
		}
	}
}

func (w *CleanupWorker) runOnce(ctx context.Context) {
	userIDs, err := w.redisSvc.ScanActiveSessions(ctx)
	if err != nil {
		log.Error().Err(err).Msg("scan active sessions failed")
		return
	}
	now := time.Now()
	for _, uid := range userIDs {
		data, err := w.redisSvc.GetActiveSession(ctx, uid)
		if err != nil || data == nil {
			continue
		}
		if now.Sub(data.LastSeen) > w.idleTimeout {
			log.Info().Str("user_id", uid.String()).Str("session_id", data.SessionID).Msg("closing idle session")
			sid, err := uuid.Parse(data.SessionID)
			if err != nil {
				_ = w.redisSvc.DeleteActiveSession(ctx, uid)
				continue
			}
			_ = w.trackingRepo.EndSession(ctx, sid, "disconnected")
			_ = w.redisSvc.DeleteActiveSession(ctx, uid)
			_ = w.redisSvc.DeleteWSConn(ctx, uid)
			_ = w.trackingRepo.UpsertDailySummary(ctx, uid, time.Now())
		}
	}
}
