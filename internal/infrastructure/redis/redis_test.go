package redis

import (
	"context"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"

	"supply-chain-simulator/internal/domain"
	"supply-chain-simulator/internal/usecase"
)

func TestStoresAndEventBus(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	defer mr.Close()

	client := NewClient(Config{Addr: mr.Addr()})
	defer client.Close()

	ctx := context.Background()
	ttl := time.Hour

	roomStore := NewRoomStore(client, ttl)
	sessionStore := NewSessionStore(client, ttl)
	decisionStore := NewDecisionStore(client, ttl)
	eventBus := NewRoomEventBus(client)

	room := domain.Room{ID: "room-1", MaxWeeks: 30}
	if err := roomStore.Save(ctx, room); err != nil {
		t.Fatalf("roomStore.Save() error = %v", err)
	}
	storedRoom, err := roomStore.GetByID(ctx, room.ID)
	if err != nil {
		t.Fatalf("roomStore.GetByID() error = %v", err)
	}
	if storedRoom.ID != room.ID {
		t.Fatalf("stored room id = %s, want %s", storedRoom.ID, room.ID)
	}

	session := domain.GameSession{ID: "session-1", RoomID: room.ID}
	if err := sessionStore.Save(ctx, session); err != nil {
		t.Fatalf("sessionStore.Save() error = %v", err)
	}
	storedSession, err := sessionStore.GetByRoomID(ctx, room.ID)
	if err != nil {
		t.Fatalf("sessionStore.GetByRoomID() error = %v", err)
	}
	if storedSession.ID != session.ID {
		t.Fatalf("stored session id = %s, want %s", storedSession.ID, session.ID)
	}

	decisions := usecase.WeeklyDecisions{
		RoomID: room.ID,
		Week:   1,
		Orders: map[domain.Role]int{domain.RoleRetailer: 4},
	}
	if err := decisionStore.Save(ctx, decisions); err != nil {
		t.Fatalf("decisionStore.Save() error = %v", err)
	}
	storedDecisions, err := decisionStore.GetByRoomAndWeek(ctx, room.ID, 1)
	if err != nil {
		t.Fatalf("decisionStore.GetByRoomAndWeek() error = %v", err)
	}
	if storedDecisions.Orders[domain.RoleRetailer] != 4 {
		t.Fatalf("stored order = %d, want 4", storedDecisions.Orders[domain.RoleRetailer])
	}
	if err := decisionStore.DeleteByRoomAndWeek(ctx, room.ID, 1); err != nil {
		t.Fatalf("decisionStore.DeleteByRoomAndWeek() error = %v", err)
	}

	events, cancel := eventBus.Subscribe(room.ID)
	defer cancel()

	wantEvent := usecase.RoomEvent{Type: "room.updated", RoomID: room.ID}
	if err := eventBus.PublishRoomEvent(ctx, wantEvent); err != nil {
		t.Fatalf("eventBus.PublishRoomEvent() error = %v", err)
	}

	select {
	case got := <-events:
		if got.Type != wantEvent.Type {
			t.Fatalf("event type = %s, want %s", got.Type, wantEvent.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for redis room event")
	}
}
