package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RedisService struct {
	client *redis.Client
}

func NewRedisService(client *redis.Client) *RedisService {
	return &RedisService{client: client}
}

// Keys
func activeSessionKey(userID uuid.UUID) string { return fmt.Sprintf("session:active:%s", userID.String()) }
func pointsBufferKey(sessionID uuid.UUID) string { return fmt.Sprintf("points:buffer:%s", sessionID.String()) }
func wsConnKey(userID uuid.UUID) string { return fmt.Sprintf("ws:conn:%s", userID.String()) }

const forceDisconnectChannel = "ws:force_disconnect"

// Active session

type ActiveSessionData struct {
	SessionID string    `json:"session_id"`
	LastSeen  time.Time `json:"last_seen"`
}

func (s *RedisService) SetActiveSession(ctx context.Context, userID, sessionID uuid.UUID) error {
	data := ActiveSessionData{SessionID: sessionID.String(), LastSeen: time.Now()}
	b, _ := json.Marshal(data)
	return s.client.Set(ctx, activeSessionKey(userID), string(b), 0).Err()
}

func (s *RedisService) GetActiveSession(ctx context.Context, userID uuid.UUID) (*ActiveSessionData, error) {
	val, err := s.client.Get(ctx, activeSessionKey(userID)).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var d ActiveSessionData
	if err := json.Unmarshal([]byte(val), &d); err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *RedisService) UpdateLastSeen(ctx context.Context, userID uuid.UUID) error {
	data, err := s.GetActiveSession(ctx, userID)
	if err != nil || data == nil {
		return err
	}
	data.LastSeen = time.Now()
	b, _ := json.Marshal(data)
	return s.client.Set(ctx, activeSessionKey(userID), string(b), 0).Err()
}

func (s *RedisService) DeleteActiveSession(ctx context.Context, userID uuid.UUID) error {
	return s.client.Del(ctx, activeSessionKey(userID)).Err()
}

// Points buffer

func (s *RedisService) PushPoints(ctx context.Context, sessionID uuid.UUID, pointsJSON string) error {
	return s.client.RPush(ctx, pointsBufferKey(sessionID), pointsJSON).Err()
}

func (s *RedisService) PopAllPoints(ctx context.Context, sessionID uuid.UUID) ([]string, error) {
	key := pointsBufferKey(sessionID)
	// Use LRange + Del atomically via pipeline
	vals, err := s.client.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, err
	}
	if len(vals) > 0 {
		_ = s.client.Del(ctx, key).Err()
	}
	return vals, nil
}

func (s *RedisService) BufferLen(ctx context.Context, sessionID uuid.UUID) (int64, error) {
	return s.client.LLen(ctx, pointsBufferKey(sessionID)).Result()
}

// WS conn tracking

func (s *RedisService) SetWSConn(ctx context.Context, userID uuid.UUID, instanceID string, ttl time.Duration) error {
	return s.client.Set(ctx, wsConnKey(userID), instanceID, ttl).Err()
}

func (s *RedisService) DeleteWSConn(ctx context.Context, userID uuid.UUID) error {
	return s.client.Del(ctx, wsConnKey(userID)).Err()
}

// Pub/Sub for force disconnect

func (s *RedisService) PublishForceDisconnect(ctx context.Context, userID uuid.UUID) error {
	return s.client.Publish(ctx, forceDisconnectChannel, userID.String()).Err()
}

func (s *RedisService) SubscribeForceDisconnect(ctx context.Context) *redis.PubSub {
	return s.client.Subscribe(ctx, forceDisconnectChannel)
}

// Scan active sessions for cleanup
func (s *RedisService) ScanActiveSessions(ctx context.Context) ([]uuid.UUID, error) {
	var userIDs []uuid.UUID
	iter := s.client.Scan(ctx, 0, "session:active:*", 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		// key = session:active:{uuid}
		idStr := key[len("session:active:"):]
		if id, err := uuid.Parse(idStr); err == nil {
			userIDs = append(userIDs, id)
		}
	}
	return userIDs, iter.Err()
}

func (s *RedisService) Client() *redis.Client { return s.client }
