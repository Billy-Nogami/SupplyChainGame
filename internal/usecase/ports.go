package usecase

import (
	"context"
	"time"

	"supply-chain-simulator/internal/domain"
)

type RoomStore interface {
	Save(ctx context.Context, room domain.Room) error
	GetByID(ctx context.Context, roomID string) (domain.Room, error)
}

type IDGenerator interface {
	NewID() (string, error)
}

type Clock interface {
	Now() time.Time
}
