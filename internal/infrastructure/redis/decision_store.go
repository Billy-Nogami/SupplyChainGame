package redis

import (
	"context"
	"encoding/json"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"supply-chain-simulator/internal/domain"
	"supply-chain-simulator/internal/usecase"
)

type DecisionStore struct {
	client *goredis.Client
	ttl    time.Duration
}

func NewDecisionStore(client *goredis.Client, ttl time.Duration) *DecisionStore {
	return &DecisionStore{client: client, ttl: ttl}
}

func (s *DecisionStore) Save(ctx context.Context, decisions usecase.WeeklyDecisions) error {
	payload, err := json.Marshal(decisions)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, decisionsKey(decisions.RoomID, decisions.Week), payload, s.ttl).Err()
}

func (s *DecisionStore) GetByRoomAndWeek(ctx context.Context, roomID string, week int) (usecase.WeeklyDecisions, error) {
	payload, err := s.client.Get(ctx, decisionsKey(roomID, week)).Bytes()
	if err == goredis.Nil {
		return usecase.WeeklyDecisions{}, domain.ErrWeekDecisionsNotFound
	}
	if err != nil {
		return usecase.WeeklyDecisions{}, err
	}

	var decisions usecase.WeeklyDecisions
	if err := json.Unmarshal(payload, &decisions); err != nil {
		return usecase.WeeklyDecisions{}, err
	}

	return decisions, nil
}

func (s *DecisionStore) DeleteByRoomAndWeek(ctx context.Context, roomID string, week int) error {
	return s.client.Del(ctx, decisionsKey(roomID, week)).Err()
}
