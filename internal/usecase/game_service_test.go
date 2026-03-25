package usecase

import (
	"context"
	"testing"
	"time"

	"supply-chain-simulator/internal/domain"
	"supply-chain-simulator/internal/infrastructure/memory"
)

func TestStartGameCreatesSessionAndAssignsRoles(t *testing.T) {
	roomStore := memory.NewRoomStore()
	sessionStore := memory.NewSessionStore()
	idGenerator := &stubIDGenerator{ids: []string{"room-1", "player-1", "player-2", "player-3", "player-4", "session-1"}}
	clock := stubClock{now: time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)}

	roomService := NewRoomService(roomStore, idGenerator, clock)
	room, err := roomService.CreateRoom(context.Background(), 30)
	if err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}

	playerNames := []string{"Alice", "Bob", "Charlie", "Dana"}
	for _, name := range playerNames {
		room, err = roomService.JoinRoom(context.Background(), room.ID, name)
		if err != nil {
			t.Fatalf("JoinRoom(%q) error = %v", name, err)
		}
	}

	gameService := NewGameService(roomStore, sessionStore, idGenerator, clock)
	session, err := gameService.StartGame(context.Background(), room.ID, testScenario())
	if err != nil {
		t.Fatalf("StartGame() error = %v", err)
	}

	if session.Status != domain.GameStatusActive {
		t.Fatalf("session status = %s, want %s", session.Status, domain.GameStatusActive)
	}
	if len(session.Nodes) != len(domain.AllRoles) {
		t.Fatalf("nodes length = %d, want %d", len(session.Nodes), len(domain.AllRoles))
	}

	updatedRoom, err := roomStore.GetByID(context.Background(), room.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if updatedRoom.Status != domain.GameStatusActive {
		t.Fatalf("room status = %s, want %s", updatedRoom.Status, domain.GameStatusActive)
	}

	roleSet := make(map[domain.Role]bool, len(updatedRoom.Players))
	for _, player := range updatedRoom.Players {
		if player.Role == "" {
			t.Fatal("player role is empty, want assigned role")
		}
		roleSet[player.Role] = true
	}
	if len(roleSet) != len(domain.AllRoles) {
		t.Fatalf("unique assigned roles = %d, want %d", len(roleSet), len(domain.AllRoles))
	}
}

type stubIDGenerator struct {
	ids []string
	pos int
}

func (s *stubIDGenerator) NewID() (string, error) {
	value := s.ids[s.pos]
	s.pos++
	return value, nil
}

type stubClock struct {
	now time.Time
}

func (s stubClock) Now() time.Time {
	return s.now
}

func testScenario() domain.Scenario {
	return domain.Scenario{
		ID:                    "scenario-1",
		InitialInventory:      12,
		InitialBacklog:        0,
		InitialPipelineGoods:  []int{4, 4},
		InitialPipelineOrders: []int{4},
		ConsumerDemand:        []int{5, 5, 5, 5, 9, 9},
		ShippingDelay:         2,
		OrderDelay:            1,
		ProductionDelay:       2,
		HoldingCost:           1,
		BacklogCost:           2,
	}
}
