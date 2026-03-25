package memory

import (
	"context"
	"sync"

	"supply-chain-simulator/internal/domain"
)

type RoomStore struct {
	mu    sync.RWMutex
	rooms map[string]domain.Room
}

func NewRoomStore() *RoomStore {
	return &RoomStore{
		rooms: make(map[string]domain.Room),
	}
}

func (s *RoomStore) Save(_ context.Context, room domain.Room) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.rooms[room.ID] = room

	return nil
}

func (s *RoomStore) GetByID(_ context.Context, roomID string) (domain.Room, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	room, ok := s.rooms[roomID]
	if !ok {
		return domain.Room{}, domain.ErrRoomNotFound
	}

	return room, nil
}
