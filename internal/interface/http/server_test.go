package http

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"supply-chain-simulator/internal/domain"
	infraexport "supply-chain-simulator/internal/infrastructure/export"
	"supply-chain-simulator/internal/infrastructure/memory"
	"supply-chain-simulator/internal/usecase"
)

func TestServerRoomGameplayFlow(t *testing.T) {
	server := newTestServer()

	room := createRoom(t, server)

	var players []domain.Player
	for _, name := range []string{"Alice", "Bob", "Charlie", "Dana"} {
		room = joinRoom(t, server, room.ID, name)
		players = room.Players
	}

	for i, player := range players {
		assignRole(t, server, room.ID, player.ID, domain.AllRoles[i])
	}

	startedSession := startGame(t, server, room.ID)
	if startedSession.RoomID != room.ID {
		t.Fatalf("session room id = %s, want %s", startedSession.RoomID, room.ID)
	}

	for i, player := range players {
		snapshot := submitOrder(t, server, room.ID, player.ID, 4)
		if i == len(players)-1 && !snapshot.Ready {
			t.Fatal("snapshot.Ready = false, want true")
		}
	}

	weekState := advanceWeek(t, server, room.ID)
	if weekState.Week != 1 {
		t.Fatalf("week state week = %d, want 1", weekState.Week)
	}

	weeks := getWeeks(t, server, room.ID)
	if len(weeks) != 1 {
		t.Fatalf("weeks length = %d, want 1", len(weeks))
	}

	analytics := getAnalytics(t, server, room.ID)
	if analytics.TotalCost != 48 {
		t.Fatalf("analytics total cost = %d, want 48", analytics.TotalCost)
	}

	exported := exportSession(t, server, room.ID)
	if len(exported) == 0 {
		t.Fatal("exported file is empty")
	}
	if string(exported[:2]) != "PK" {
		t.Fatalf("export prefix = %q, want zip header PK", string(exported[:2]))
	}

	decisions := getDecisions(t, server, room.ID)
	if decisions.Week != 2 {
		t.Fatalf("decisions week = %d, want 2", decisions.Week)
	}
}

func TestWithCORSPrefight(t *testing.T) {
	handler := WithCORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), []string{"http://localhost:3000"})

	req := httptest.NewRequest(http.MethodOptions, "/rooms", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatalf("allow origin = %q, want %q", rec.Header().Get("Access-Control-Allow-Origin"), "http://localhost:3000")
	}
}

func TestRoomEventsStreamReturnsInitialSnapshot(t *testing.T) {
	server := newTestServer()
	room := createRoom(t, server)

	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	resp, err := http.Get(testServer.URL + "/rooms/" + room.ID + "/events")
	if err != nil {
		t.Fatalf("http.Get(events) error = %v", err)
	}
	defer resp.Body.Close()

	if contentType := resp.Header.Get("Content-Type"); !strings.Contains(contentType, "text/event-stream") {
		t.Fatalf("content-type = %q, want text/event-stream", contentType)
	}

	reader := bufio.NewReader(resp.Body)
	eventLine := readLine(t, reader)
	dataLine := readLine(t, reader)

	if eventLine != "event: room.snapshot" {
		t.Fatalf("event line = %q, want %q", eventLine, "event: room.snapshot")
	}
	if !strings.Contains(dataLine, "\"room_id\":\""+room.ID+"\"") {
		t.Fatalf("data line = %q, want room id payload", dataLine)
	}
}

func newTestServer() *Server {
	roomStore := memory.NewRoomStore()
	sessionStore := memory.NewSessionStore()
	decisionStore := memory.NewDecisionStore()
	scenarioRepo := memory.NewScenarioRepository()
	eventBus := memory.NewRoomEventBus()
	idGenerator := &stubIDGenerator{ids: []string{"room-1", "player-1", "player-2", "player-3", "player-4", "session-1"}}
	clock := stubClock{now: time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)}
	exporter := infraexport.NewXLSXExporter()

	roomService := usecase.NewRoomService(roomStore, idGenerator, clock, eventBus)
	gameService := usecase.NewGameService(roomStore, sessionStore, decisionStore, scenarioRepo, exporter, eventBus, idGenerator, clock)

	return NewServer(roomService, gameService, eventBus)
}

func createRoom(t *testing.T, server *Server) domain.Room {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/rooms", bytes.NewBufferString(`{"max_weeks":30}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create room status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var room domain.Room
	if err := json.Unmarshal(rec.Body.Bytes(), &room); err != nil {
		t.Fatalf("json.Unmarshal(room) error = %v", err)
	}

	return room
}

func joinRoom(t *testing.T, server *Server, roomID, name string) domain.Room {
	t.Helper()

	body, _ := json.Marshal(map[string]string{"name": name})
	req := httptest.NewRequest(http.MethodPost, "/rooms/"+roomID+"/players", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("join room status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var room domain.Room
	if err := json.Unmarshal(rec.Body.Bytes(), &room); err != nil {
		t.Fatalf("json.Unmarshal(join room) error = %v", err)
	}

	return room
}

func assignRole(t *testing.T, server *Server, roomID, playerID string, role domain.Role) {
	t.Helper()

	body, _ := json.Marshal(map[string]string{
		"player_id": playerID,
		"role":      string(role),
	})
	req := httptest.NewRequest(http.MethodPost, "/rooms/"+roomID+"/roles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("assign role status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func startGame(t *testing.T, server *Server, roomID string) domain.GameSession {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/rooms/"+roomID+"/start", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("start game status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var session domain.GameSession
	if err := json.Unmarshal(rec.Body.Bytes(), &session); err != nil {
		t.Fatalf("json.Unmarshal(session) error = %v", err)
	}

	return session
}

func submitOrder(t *testing.T, server *Server, roomID, playerID string, order int) usecase.WeeklyDecisionsSnapshot {
	t.Helper()

	body, _ := json.Marshal(map[string]any{
		"player_id": playerID,
		"order":     order,
	})
	req := httptest.NewRequest(http.MethodPost, "/rooms/"+roomID+"/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("submit order status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var snapshot usecase.WeeklyDecisionsSnapshot
	if err := json.Unmarshal(rec.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("json.Unmarshal(decisions) error = %v", err)
	}

	return snapshot
}

func advanceWeek(t *testing.T, server *Server, roomID string) domain.WeekState {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/rooms/"+roomID+"/next", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("advance week status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var weekState domain.WeekState
	if err := json.Unmarshal(rec.Body.Bytes(), &weekState); err != nil {
		t.Fatalf("json.Unmarshal(weekState) error = %v", err)
	}

	return weekState
}

func getWeeks(t *testing.T, server *Server, roomID string) []domain.WeekState {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/rooms/"+roomID+"/weeks", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get weeks status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var weeks []domain.WeekState
	if err := json.Unmarshal(rec.Body.Bytes(), &weeks); err != nil {
		t.Fatalf("json.Unmarshal(weeks) error = %v", err)
	}

	return weeks
}

func getDecisions(t *testing.T, server *Server, roomID string) usecase.WeeklyDecisionsSnapshot {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/rooms/"+roomID+"/decisions", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get decisions status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var snapshot usecase.WeeklyDecisionsSnapshot
	if err := json.Unmarshal(rec.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("json.Unmarshal(decisions) error = %v", err)
	}

	return snapshot
}

func getAnalytics(t *testing.T, server *Server, roomID string) domain.SessionAnalytics {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/rooms/"+roomID+"/analytics", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get analytics status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var analytics domain.SessionAnalytics
	if err := json.Unmarshal(rec.Body.Bytes(), &analytics); err != nil {
		t.Fatalf("json.Unmarshal(analytics) error = %v", err)
	}

	return analytics
}

func exportSession(t *testing.T, server *Server, roomID string) []byte {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/rooms/"+roomID+"/export", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("export status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	return rec.Body.Bytes()
}

func readLine(t *testing.T, reader *bufio.Reader) string {
	t.Helper()

	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		t.Fatalf("ReadString() error = %v", err)
	}

	return strings.TrimRight(line, "\n")
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
