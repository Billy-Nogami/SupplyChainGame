package redis

import (
	"context"
	"encoding/json"
	"sync"

	goredis "github.com/redis/go-redis/v9"

	"supply-chain-simulator/internal/usecase"
)

type RoomEventBus struct {
	client *goredis.Client
}

func NewRoomEventBus(client *goredis.Client) *RoomEventBus {
	return &RoomEventBus{client: client}
}

func (b *RoomEventBus) PublishRoomEvent(ctx context.Context, event usecase.RoomEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return b.client.Publish(ctx, roomEventsChannel(event.RoomID), payload).Err()
}

func (b *RoomEventBus) Subscribe(roomID string) (<-chan usecase.RoomEvent, func()) {
	pubsub := b.client.Subscribe(context.Background(), roomEventsChannel(roomID))
	out := make(chan usecase.RoomEvent, 16)

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			_ = pubsub.Close()
			close(out)
		})
	}

	go func() {
		defer cancel()
		if _, err := pubsub.Receive(context.Background()); err != nil {
			return
		}
		ch := pubsub.Channel()
		for message := range ch {
			var event usecase.RoomEvent
			if err := json.Unmarshal([]byte(message.Payload), &event); err != nil {
				continue
			}

			select {
			case out <- event:
			default:
			}
		}
	}()

	return out, cancel
}
