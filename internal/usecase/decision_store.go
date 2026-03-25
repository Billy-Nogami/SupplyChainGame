package usecase

import (
	"context"
	"time"

	"supply-chain-simulator/internal/domain"
)

type WeeklyDecisions struct {
	RoomID    string
	Week      int
	Orders    map[domain.Role]int
	UpdatedAt time.Time
}

func (w WeeklyDecisions) Snapshot() WeeklyDecisionsSnapshot {
	submitted := make([]domain.Role, 0, len(w.Orders))
	pending := make([]domain.Role, 0, len(domain.AllRoles))
	for _, role := range domain.AllRoles {
		if _, ok := w.Orders[role]; ok {
			submitted = append(submitted, role)
			continue
		}
		pending = append(pending, role)
	}

	orders := make(map[domain.Role]int, len(w.Orders))
	for role, order := range w.Orders {
		orders[role] = order
	}

	return WeeklyDecisionsSnapshot{
		RoomID:         w.RoomID,
		Week:           w.Week,
		Orders:         orders,
		SubmittedRoles: submitted,
		PendingRoles:   pending,
		Ready:          len(pending) == 0,
	}
}

type WeeklyDecisionsSnapshot struct {
	RoomID         string
	Week           int
	Orders         map[domain.Role]int
	SubmittedRoles []domain.Role
	PendingRoles   []domain.Role
	Ready          bool
}

type DecisionStore interface {
	Save(ctx context.Context, decisions WeeklyDecisions) error
	GetByRoomAndWeek(ctx context.Context, roomID string, week int) (WeeklyDecisions, error)
	DeleteByRoomAndWeek(ctx context.Context, roomID string, week int) error
}
