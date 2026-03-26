package redis

import (
	"context"
	"encoding/json"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"supply-chain-simulator/internal/domain"
)

type RoomStore struct {
	client *goredis.Client
	ttl    time.Duration
}

func NewRoomStore(client *goredis.Client, ttl time.Duration) *RoomStore {
	return &RoomStore{client: client, ttl: ttl}
}

func (s *RoomStore) Save(ctx context.Context, room domain.Room) error {
	payload, err := json.Marshal(room)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, roomKey(room.ID), payload, s.ttl).Err()
}

func (s *RoomStore) GetByID(ctx context.Context, roomID string) (domain.Room, error) {
	payload, err := s.client.Get(ctx, roomKey(roomID)).Bytes()
	if err == goredis.Nil {
		return domain.Room{}, domain.ErrRoomNotFound
	}
	if err != nil {
		return domain.Room{}, err
	}

	var room domain.Room
	if err := json.Unmarshal(payload, &room); err != nil {
		return domain.Room{}, err
	}

	return room, nil
}
