package redis

import (
	"context"
	"encoding/json"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"supply-chain-simulator/internal/domain"
)

type SessionStore struct {
	client *goredis.Client
	ttl    time.Duration
}

func NewSessionStore(client *goredis.Client, ttl time.Duration) *SessionStore {
	return &SessionStore{client: client, ttl: ttl}
}

func (s *SessionStore) Save(ctx context.Context, session domain.GameSession) error {
	payload, err := json.Marshal(session)
	if err != nil {
		return err
	}

	pipe := s.client.TxPipeline()
	pipe.Set(ctx, sessionKey(session.ID), payload, s.ttl)
	pipe.Set(ctx, roomSessionKey(session.RoomID), session.ID, s.ttl)
	_, err = pipe.Exec(ctx)

	return err
}

func (s *SessionStore) GetByID(ctx context.Context, sessionID string) (domain.GameSession, error) {
	payload, err := s.client.Get(ctx, sessionKey(sessionID)).Bytes()
	if err == goredis.Nil {
		return domain.GameSession{}, domain.ErrSessionNotFound
	}
	if err != nil {
		return domain.GameSession{}, err
	}

	var session domain.GameSession
	if err := json.Unmarshal(payload, &session); err != nil {
		return domain.GameSession{}, err
	}

	return session, nil
}

func (s *SessionStore) GetByRoomID(ctx context.Context, roomID string) (domain.GameSession, error) {
	sessionID, err := s.client.Get(ctx, roomSessionKey(roomID)).Result()
	if err == goredis.Nil {
		return domain.GameSession{}, domain.ErrSessionNotFound
	}
	if err != nil {
		return domain.GameSession{}, err
	}

	return s.GetByID(ctx, sessionID)
}
