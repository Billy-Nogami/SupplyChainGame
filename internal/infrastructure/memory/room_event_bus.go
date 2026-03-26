package memory

import (
	"context"
	"sync"

	"supply-chain-simulator/internal/usecase"
)

type RoomEventBus struct {
	mu          sync.RWMutex
	subscribers map[string]map[int]chan usecase.RoomEvent
	nextID      int
}

func NewRoomEventBus() *RoomEventBus {
	return &RoomEventBus{
		subscribers: make(map[string]map[int]chan usecase.RoomEvent),
	}
}

func (b *RoomEventBus) PublishRoomEvent(_ context.Context, event usecase.RoomEvent) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, subscriber := range b.subscribers[event.RoomID] {
		select {
		case subscriber <- event:
		default:
		}
	}

	return nil
}

func (b *RoomEventBus) Subscribe(roomID string) (<-chan usecase.RoomEvent, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.nextID++
	id := b.nextID
	ch := make(chan usecase.RoomEvent, 16)

	if _, ok := b.subscribers[roomID]; !ok {
		b.subscribers[roomID] = make(map[int]chan usecase.RoomEvent)
	}
	b.subscribers[roomID][id] = ch

	cancel := func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		roomSubscribers, ok := b.subscribers[roomID]
		if !ok {
			return
		}

		if existing, ok := roomSubscribers[id]; ok {
			delete(roomSubscribers, id)
			close(existing)
		}
		if len(roomSubscribers) == 0 {
			delete(b.subscribers, roomID)
		}
	}

	return ch, cancel
}
