package memory

import (
	"context"
	"fmt"
	"sync"

	"supply-chain-simulator/internal/domain"
	"supply-chain-simulator/internal/usecase"
)

type DecisionStore struct {
	mu        sync.RWMutex
	decisions map[string]usecase.WeeklyDecisions
}

func NewDecisionStore() *DecisionStore {
	return &DecisionStore{
		decisions: make(map[string]usecase.WeeklyDecisions),
	}
}

func (s *DecisionStore) Save(_ context.Context, decisions usecase.WeeklyDecisions) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	copiedOrders := make(map[domain.Role]int, len(decisions.Orders))
	for role, order := range decisions.Orders {
		copiedOrders[role] = order
	}

	decisions.Orders = copiedOrders
	s.decisions[key(decisions.RoomID, decisions.Week)] = decisions

	return nil
}

func (s *DecisionStore) GetByRoomAndWeek(_ context.Context, roomID string, week int) (usecase.WeeklyDecisions, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	decisions, ok := s.decisions[key(roomID, week)]
	if !ok {
		return usecase.WeeklyDecisions{}, domain.ErrWeekDecisionsNotFound
	}

	copiedOrders := make(map[domain.Role]int, len(decisions.Orders))
	for role, order := range decisions.Orders {
		copiedOrders[role] = order
	}

	decisions.Orders = copiedOrders

	return decisions, nil
}

func (s *DecisionStore) DeleteByRoomAndWeek(_ context.Context, roomID string, week int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.decisions, key(roomID, week))

	return nil
}

func key(roomID string, week int) string {
	return fmt.Sprintf("%s:%d", roomID, week)
}
