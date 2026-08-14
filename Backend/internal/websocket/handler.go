package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"github.com/vmihailenco/msgpack/v5"
	"tracking-backend/internal/auth"
	"tracking-backend/internal/config"
	"tracking-backend/internal/models"
	"tracking-backend/internal/repository"
	"tracking-backend/internal/service"
	"tracking-backend/internal/utils"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool { return true },
}

type WSHandler struct {
	cfg         *config.Config
	trackingRepo *repository.TrackingRepo
	redisSvc    *service.RedisService
	instanceID  string

	// active connections
	mu    sync.RWMutex
	conns map[uuid.UUID]*websocket.Conn
}

func NewWSHandler(cfg *config.Config, trackingRepo *repository.TrackingRepo, redisSvc *service.RedisService) *WSHandler {
	return &WSHandler{
		cfg:          cfg,
		trackingRepo: trackingRepo,
		redisSvc:     redisSvc,
		instanceID:   uuid.NewString(),
		conns:        make(map[uuid.UUID]*websocket.Conn),
	}
}

func (h *WSHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// Auth via query param token or Authorization header
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		// Try Authorization header
		authHeader := r.Header.Get("Authorization")
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenStr = authHeader[7:]
		}
	}
	if tokenStr == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}
	claims, err := auth.ParseAccessToken(h.cfg, tokenStr)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}
	userID := claims.UserID

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("websocket upgrade failed")
		return
	}
	defer conn.Close()

	// Register connection
	h.mu.Lock()
	// Close existing connection for same user (single device per PRD)
	if old, ok := h.conns[userID]; ok {
		_ = old.Close()
	}
	h.conns[userID] = conn
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.conns, userID)
		h.mu.Unlock()
		_ = h.redisSvc.DeleteWSConn(context.Background(), userID)
	}()

	// Ensure active session exists
	ctx := context.Background()
	active, err := h.trackingRepo.GetActiveSession(ctx, userID)
	if err != nil {
		log.Error().Err(err).Msg("get active session failed")
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","message":"internal error"}`))
		return
	}
	if active == nil {
		now := time.Now()
		active = &models.TrackingSession{
			ID:            uuid.New(),
			UserID:        userID,
			StartedAt:     now,
			Status:        "active",
			TotalDistance: 0,
			SessionDate:   time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC),
		}
		if err := h.trackingRepo.CreateSession(ctx, active); err != nil {
			log.Error().Err(err).Msg("create session failed")
			return
		}
	}
	_ = h.redisSvc.SetActiveSession(ctx, userID, active.ID)
	_ = h.redisSvc.SetWSConn(ctx, userID, h.instanceID, 0)

	// Send welcome with session_id
	welcome := map[string]string{"type": "connected", "session_id": active.ID.String()}
	if b, err := json.Marshal(welcome); err == nil {
		_ = conn.WriteMessage(websocket.TextMessage, b)
	}

	// Set read deadline with pong handler
	conn.SetReadDeadline(time.Now().Add(h.cfg.WSIdleTimeout + 30*time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(h.cfg.WSIdleTimeout + 30*time.Second))
		return nil
	})

	// Ping ticker
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			h.mu.RLock()
			_, ok := h.conns[userID]
			h.mu.RUnlock()
			if !ok {
				return
			}
			_ = conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second))
		}
	}()

	// Read loop
	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Warn().Err(err).Str("user_id", userID.String()).Msg("ws read error")
			}
			break
		}

		var msg models.WSMessage
		if msgType == websocket.BinaryMessage {
			if err := msgpack.Unmarshal(data, &msg); err != nil {
				// Try JSON fallback
				if err2 := json.Unmarshal(data, &msg); err2 != nil {
					log.Warn().Err(err).Msg("failed to decode ws message")
					continue
				}
			}
		} else {
			if err := json.Unmarshal(data, &msg); err != nil {
				// Try msgpack fallback
				if err2 := msgpack.Unmarshal(data, &msg); err2 != nil {
					log.Warn().Err(err).Msg("failed to decode ws text message")
					continue
				}
			}
		}

		switch msg.Type {
		case "location_update":
			h.handleLocationUpdate(ctx, userID, active.ID, &msg, conn)
		case "heartbeat":
			_ = h.redisSvc.UpdateLastSeen(ctx, userID)
			// ACK heartbeat
			ack := models.WSAck{Type: "ack", Status: "ok", Message: "heartbeat"}
			if b, err := msgpack.Marshal(ack); err == nil {
				_ = conn.WriteMessage(websocket.BinaryMessage, b)
			}
		case "session_end":
			_ = h.trackingRepo.EndSession(ctx, active.ID, "ended")
			_ = h.redisSvc.DeleteActiveSession(ctx, userID)
			ack := models.WSAck{Type: "ack", Status: "ok", SessionID: active.ID.String(), Message: "session ended"}
			if b, err := msgpack.Marshal(ack); err == nil {
				_ = conn.WriteMessage(websocket.BinaryMessage, b)
			}
			return
		default:
			log.Warn().Str("type", msg.Type).Msg("unknown ws message type")
		}

		// Refresh read deadline
		conn.SetReadDeadline(time.Now().Add(h.cfg.WSIdleTimeout + 30*time.Second))
	}
}

func (h *WSHandler) handleLocationUpdate(ctx context.Context, userID, sessionID uuid.UUID, msg *models.WSMessage, conn *websocket.Conn) {
	if msg.Lat == nil || msg.Lng == nil {
		return
	}
	lat, lng := *msg.Lat, *msg.Lng
	if !utils.IsValidLatLng(lat, lng) {
		return
	}

	// Parse timestamp
	recordedAt := time.Now()
	if msg.Timestamp != nil && *msg.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339, *msg.Timestamp); err == nil {
			recordedAt = t
		}
	}

	// Resolve session_id from message if provided
	if msg.SessionID != "" {
		if sid, err := uuid.Parse(msg.SessionID); err == nil {
			sessionID = sid
		}
	}

	// Get last point for distance filtering
	lastPoint, _ := h.trackingRepo.GetLastPoint(ctx, sessionID)
	var dist float64
	if lastPoint != nil {
		dist = utils.Haversine(lastPoint.Lat, lastPoint.Lng, lat, lng)
		if utils.ShouldIgnorePoint(msg.Accuracy, dist) {
			return
		}
		// Distance filter is client-side (10m), but we also ensure server doesn't double-count tiny moves
		// If dist < 1m, still save but don't count distance? For now, save all valid points
	}

	// Buffer to Redis for batch flush
	pointData := map[string]interface{}{
		"lat":         lat,
		"lng":         lng,
		"accuracy":    msg.Accuracy,
		"speed":       msg.Speed,
		"altitude":    msg.Altitude,
		"recorded_at": recordedAt.Format(time.RFC3339Nano),
	}
	b, _ := json.Marshal(pointData)
	_ = h.redisSvc.PushPoints(ctx, sessionID, string(b))

	// Update last_seen
	_ = h.redisSvc.UpdateLastSeen(ctx, userID)

	// Check if buffer reached flush threshold -> flush immediately
	if n, err := h.redisSvc.BufferLen(ctx, sessionID); err == nil && n >= int64(h.cfg.WSFlushBatchSize) {
		go h.flushSession(ctx, sessionID)
	}

	// Send ACK
	ack := models.WSAck{Type: "ack", Status: "ok", SessionID: sessionID.String()}
	if ab, err := msgpack.Marshal(ack); err == nil {
		_ = conn.WriteMessage(websocket.BinaryMessage, ab)
	}
}

func (h *WSHandler) flushSession(ctx context.Context, sessionID uuid.UUID) {
	vals, err := h.redisSvc.PopAllPoints(ctx, sessionID)
	if err != nil || len(vals) == 0 {
		return
	}
	var points []models.TrackingPoint
	var totalDist float64
	var prevLat, prevLng float64
	var hasPrev bool

	// Get last persisted point for distance continuity
	lastPoint, _ := h.trackingRepo.GetLastPoint(ctx, sessionID)
	if lastPoint != nil {
		prevLat = lastPoint.Lat
		prevLng = lastPoint.Lng
		hasPrev = true
	}

	for _, v := range vals {
		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(v), &raw); err != nil {
			continue
		}
		lat, _ := raw["lat"].(float64)
		lng, _ := raw["lng"].(float64)
		recordedAtStr, _ := raw["recorded_at"].(string)
		recordedAt, _ := time.Parse(time.RFC3339Nano, recordedAtStr)
		if recordedAt.IsZero() {
			recordedAt = time.Now()
		}
		var accuracy, speed, altitude *float64
		if a, ok := raw["accuracy"].(float64); ok {
			accuracy = &a
		}
		if s, ok := raw["speed"].(float64); ok {
			speed = &s
		}
		if al, ok := raw["altitude"].(float64); ok {
			altitude = &al
		}

		if hasPrev {
			d := utils.Haversine(prevLat, prevLng, lat, lng)
			totalDist += d
		}
		points = append(points, models.TrackingPoint{
			SessionID:  sessionID,
			Lat:        lat,
			Lng:        lng,
			Accuracy:   accuracy,
			Speed:      speed,
			Altitude:   altitude,
			RecordedAt: recordedAt,
		})
		prevLat = lat
		prevLng = lng
		hasPrev = true
	}

	if len(points) > 0 {
		if err := h.trackingRepo.InsertPoints(ctx, points); err != nil {
			log.Error().Err(err).Msg("flush insert points failed")
			return
		}
		if totalDist > 0 {
			_ = h.trackingRepo.UpdateSessionDistance(ctx, sessionID, totalDist)
		}
	}
}

// FlushAll periodically flushes all buffered points
func (h *WSHandler) StartFlushWorker(ctx context.Context) {
	ticker := time.NewTicker(h.cfg.WSFlushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.flushAll(ctx)
		}
	}
}

func (h *WSHandler) flushAll(ctx context.Context) {
	// Scan all buffer keys
	client := h.redisSvc.Client()
	iter := client.Scan(ctx, 0, "points:buffer:*", 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		sidStr := key[len("points:buffer:"):]
		sid, err := uuid.Parse(sidStr)
		if err != nil {
			continue
		}
		h.flushSession(ctx, sid)
	}
}

// CloseConn force closes a user's WS connection (for logout coordination)
func (h *WSHandler) CloseConn(userID uuid.UUID) {
	h.mu.RLock()
	conn, ok := h.conns[userID]
	h.mu.RUnlock()
	if ok {
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "force disconnect"))
		_ = conn.Close()
	}
}

// ListenForceDisconnect subscribes to Redis pub/sub for multi-instance coordination
func (h *WSHandler) ListenForceDisconnect(ctx context.Context) {
	pubsub := h.redisSvc.SubscribeForceDisconnect(ctx)
	defer pubsub.Close()
	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if uid, err := uuid.Parse(msg.Payload); err == nil {
				h.CloseConn(uid)
			}
		}
	}
}
