package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"supply-chain-simulator/internal/domain"
	"supply-chain-simulator/internal/infrastructure/memory"
	"supply-chain-simulator/internal/usecase"
)

func TestStartGameCreatesSessionAndAssignsRoles(t *testing.T) {
	gameService, roomService, room := newStartedGameServices(t)

	session, err := gameService.GetSessionByRoom(context.Background(), room.ID)
	if err != nil {
		t.Fatalf("GetSessionByRoom() error = %v", err)
	}

	if session.Status != domain.GameStatusActive {
		t.Fatalf("session status = %s, want %s", session.Status, domain.GameStatusActive)
	}
	if len(session.Nodes) != len(domain.AllRoles) {
		t.Fatalf("nodes length = %d, want %d", len(session.Nodes), len(domain.AllRoles))
	}

	updatedRoom, err := roomService.GetRoom(context.Background(), room.ID)
	if err != nil {
		t.Fatalf("GetRoom() error = %v", err)
	}
	if updatedRoom.Status != domain.GameStatusActive {
		t.Fatalf("room status = %s, want %s", updatedRoom.Status, domain.GameStatusActive)
	}
	if updatedRoom.ScenarioID != "default-beer-game" {
		t.Fatalf("room scenario id = %s, want default-beer-game", updatedRoom.ScenarioID)
	}
	if updatedRoom.CurrentWeek != 1 {
		t.Fatalf("room current week = %d, want 1", updatedRoom.CurrentWeek)
	}
}

func TestSubmitOrderAndAdvanceWeek(t *testing.T) {
	gameService, roomService, room := newStartedGameServices(t)

	if _, err := gameService.SubmitOrder(context.Background(), room.ID, room.Players[0].ID, 4); err != nil {
		t.Fatalf("SubmitOrder() first error = %v", err)
	}
	if _, err := gameService.AdvanceWeek(context.Background(), room.ID); !errors.Is(err, domain.ErrWeekNotReady) {
		t.Fatalf("AdvanceWeek() early error = %v, want %v", err, domain.ErrWeekNotReady)
	}

	for i := 1; i < len(room.Players); i++ {
		snapshot, err := gameService.SubmitOrder(context.Background(), room.ID, room.Players[i].ID, 4)
		if err != nil {
			t.Fatalf("SubmitOrder() player %d error = %v", i, err)
		}
		if i == len(room.Players)-1 && !snapshot.Ready {
			t.Fatal("snapshot.Ready = false, want true")
		}
	}

	weekState, err := gameService.AdvanceWeek(context.Background(), room.ID)
	if err != nil {
		t.Fatalf("AdvanceWeek() error = %v", err)
	}

	if weekState.Week != 1 {
		t.Fatalf("week = %d, want 1", weekState.Week)
	}
	if len(weekState.Nodes) != len(domain.AllRoles) {
		t.Fatalf("nodes length = %d, want %d", len(weekState.Nodes), len(domain.AllRoles))
	}

	session, err := gameService.GetSessionByRoom(context.Background(), room.ID)
	if err != nil {
		t.Fatalf("GetSessionByRoom() error = %v", err)
	}
	if session.CurrentWeek != 1 {
		t.Fatalf("session current week = %d, want 1", session.CurrentWeek)
	}
	if len(session.History) != 1 {
		t.Fatalf("session history length = %d, want 1", len(session.History))
	}

	updatedRoom, err := roomService.GetRoom(context.Background(), room.ID)
	if err != nil {
		t.Fatalf("GetRoom() error = %v", err)
	}
	if updatedRoom.CurrentWeek != 2 {
		t.Fatalf("room current week = %d, want 2", updatedRoom.CurrentWeek)
	}

	pending, err := gameService.GetPendingDecisions(context.Background(), room.ID)
	if err != nil {
		t.Fatalf("GetPendingDecisions() error = %v", err)
	}
	if pending.Week != 2 {
		t.Fatalf("pending week = %d, want 2", pending.Week)
	}
	if len(pending.Orders) != 0 {
		t.Fatalf("pending orders length = %d, want 0", len(pending.Orders))
	}
}

func TestGetWeeksReturnsHistory(t *testing.T) {
	gameService, _, room := newStartedGameServices(t)

	for _, player := range room.Players {
		if _, err := gameService.SubmitOrder(context.Background(), room.ID, player.ID, 4); err != nil {
			t.Fatalf("SubmitOrder() error = %v", err)
		}
	}
	if _, err := gameService.AdvanceWeek(context.Background(), room.ID); err != nil {
		t.Fatalf("AdvanceWeek() error = %v", err)
	}

	weeks, err := gameService.GetWeeks(context.Background(), room.ID)
	if err != nil {
		t.Fatalf("GetWeeks() error = %v", err)
	}
	if len(weeks) != 1 {
		t.Fatalf("weeks length = %d, want 1", len(weeks))
	}
	if weeks[0].Week != 1 {
		t.Fatalf("weeks[0].Week = %d, want 1", weeks[0].Week)
	}
}

func TestGetAnalyticsReturnsCalculatedMetrics(t *testing.T) {
	gameService, _, room := newStartedGameServices(t)

	for _, player := range room.Players {
		if _, err := gameService.SubmitOrder(context.Background(), room.ID, player.ID, 4); err != nil {
			t.Fatalf("SubmitOrder() error = %v", err)
		}
	}
	if _, err := gameService.AdvanceWeek(context.Background(), room.ID); err != nil {
		t.Fatalf("AdvanceWeek() error = %v", err)
	}

	analytics, err := gameService.GetAnalytics(context.Background(), room.ID)
	if err != nil {
		t.Fatalf("GetAnalytics() error = %v", err)
	}
	if analytics.TotalWeeks != 1 {
		t.Fatalf("analytics total weeks = %d, want 1", analytics.TotalWeeks)
	}
	if analytics.TotalCost != 48 {
		t.Fatalf("analytics total cost = %d, want 48", analytics.TotalCost)
	}
	if len(analytics.NodeAnalytics) != len(domain.AllRoles) {
		t.Fatalf("node analytics length = %d, want %d", len(analytics.NodeAnalytics), len(domain.AllRoles))
	}
}

func newStartedGameServices(t *testing.T) (*usecase.GameService, *usecase.RoomService, domain.Room) {
	t.Helper()

	roomStore := memory.NewRoomStore()
	sessionStore := memory.NewSessionStore()
	decisionStore := memory.NewDecisionStore()
	scenarioRepo := memory.NewScenarioRepository()
	idGenerator := &stubIDGenerator{ids: []string{"room-1", "player-1", "player-2", "player-3", "player-4", "session-1"}}
	clock := stubClock{now: time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)}

	roomService := usecase.NewRoomService(roomStore, idGenerator, clock)
	room, err := roomService.CreateRoom(context.Background(), 30)
	if err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}

	playerNames := []string{"Alice", "Bob", "Charlie", "Dana"}
	for i, name := range playerNames {
		room, err = roomService.JoinRoom(context.Background(), room.ID, name)
		if err != nil {
			t.Fatalf("JoinRoom(%q) error = %v", name, err)
		}

		room, err = roomService.AssignRole(context.Background(), room.ID, room.Players[i].ID, domain.AllRoles[i])
		if err != nil {
			t.Fatalf("AssignRole(%q) error = %v", name, err)
		}
	}

	gameService := usecase.NewGameService(roomStore, sessionStore, decisionStore, scenarioRepo, idGenerator, clock)
	if _, err := gameService.StartGame(context.Background(), room.ID, ""); err != nil {
		t.Fatalf("StartGame() error = %v", err)
	}

	updatedRoom, err := roomService.GetRoom(context.Background(), room.ID)
	if err != nil {
		t.Fatalf("GetRoom() error = %v", err)
	}

	return gameService, roomService, updatedRoom
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
