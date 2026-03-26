package usecase

import (
	"context"
	"time"

	"supply-chain-simulator/internal/domain"
)

type RoomEvent struct {
	Type       string                   `json:"type"`
	RoomID     string                   `json:"room_id"`
	OccurredAt time.Time                `json:"occurred_at"`
	Room       *domain.Room             `json:"room,omitempty"`
	Session    *domain.GameSession      `json:"session,omitempty"`
	Decisions  *WeeklyDecisionsSnapshot `json:"decisions,omitempty"`
	Analytics  *domain.SessionAnalytics `json:"analytics,omitempty"`
}

type RoomEventPublisher interface {
	PublishRoomEvent(ctx context.Context, event RoomEvent) error
}

type NopRoomEventPublisher struct{}

func (NopRoomEventPublisher) PublishRoomEvent(_ context.Context, _ RoomEvent) error {
	return nil
}
