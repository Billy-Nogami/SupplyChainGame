package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"supply-chain-simulator/internal/domain"
)

type RoomService interface {
	CreateRoom(ctx context.Context, maxWeeks int) (domain.Room, error)
	GetRoom(ctx context.Context, roomID string) (domain.Room, error)
	JoinRoom(ctx context.Context, roomID, playerName string) (domain.Room, error)
	AssignRole(ctx context.Context, roomID, playerID string, role domain.Role) (domain.Room, error)
}

type GameService interface {
	StartGame(ctx context.Context, roomID, scenarioID string) (domain.GameSession, error)
	GetSessionByRoom(ctx context.Context, roomID string) (domain.GameSession, error)
}

type Server struct {
	roomService RoomService
	gameService GameService
	mux         *http.ServeMux
}

type createRoomRequest struct {
	MaxWeeks int `json:"max_weeks"`
}

type joinRoomRequest struct {
	Name string `json:"name"`
}

type assignRoleRequest struct {
	PlayerID string `json:"player_id"`
	Role     string `json:"role"`
}

type startGameRequest struct {
	ScenarioID string `json:"scenario_id"`
}

func NewServer(roomService RoomService, gameService GameService) *Server {
	server := &Server{
		roomService: roomService,
		gameService: gameService,
		mux:         http.NewServeMux(),
	}

	server.registerRoutes()

	return server
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealth)
	s.mux.HandleFunc("POST /rooms", s.handleCreateRoom)
	s.mux.HandleFunc("GET /rooms/", s.handleGetRoomResource)
	s.mux.HandleFunc("POST /rooms/", s.handleRoomCommand)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	var req createRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.MaxWeeks == 0 {
		req.MaxWeeks = 30
	}

	room, err := s.roomService.CreateRoom(r.Context(), req.MaxWeeks)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, room)
}

func (s *Server) handleGetRoomResource(w http.ResponseWriter, r *http.Request) {
	roomID, action := parseRoomPath(r.URL.Path)
	if roomID == "" {
		http.NotFound(w, r)
		return
	}

	switch action {
	case "":
		room, err := s.roomService.GetRoom(r.Context(), roomID)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, room)
	case "session":
		session, err := s.gameService.GetSessionByRoom(r.Context(), roomID)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, session)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleRoomCommand(w http.ResponseWriter, r *http.Request) {
	roomID, action := parseRoomPath(r.URL.Path)
	if roomID == "" {
		http.NotFound(w, r)
		return
	}

	switch action {
	case "players":
		s.handleJoinRoom(w, r, roomID)
	case "roles":
		s.handleAssignRole(w, r, roomID)
	case "start":
		s.handleStartGame(w, r, roomID)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleJoinRoom(w http.ResponseWriter, r *http.Request, roomID string) {
	var req joinRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	room, err := s.roomService.JoinRoom(r.Context(), roomID, req.Name)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, room)
}

func (s *Server) handleAssignRole(w http.ResponseWriter, r *http.Request, roomID string) {
	var req assignRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	room, err := s.roomService.AssignRole(r.Context(), roomID, req.PlayerID, domain.Role(req.Role))
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, room)
}

func (s *Server) handleStartGame(w http.ResponseWriter, r *http.Request, roomID string) {
	var req startGameRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}

	session, err := s.gameService.StartGame(r.Context(), roomID, req.ScenarioID)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, session)
}

func parseRoomPath(path string) (roomID, action string) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) < 2 || parts[0] != "rooms" {
		return "", ""
	}
	if len(parts) == 2 {
		return parts[1], ""
	}
	if len(parts) == 3 {
		return parts[1], parts[2]
	}

	return "", ""
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrRoomNotFound),
		errors.Is(err, domain.ErrSessionNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrInvalidMaxWeeks),
		errors.Is(err, domain.ErrEmptyPlayerName),
		errors.Is(err, domain.ErrPlayerAlreadyIn),
		errors.Is(err, domain.ErrPlayerNotFound),
		errors.Is(err, domain.ErrInvalidRole):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrRoomFull),
		errors.Is(err, domain.ErrCannotJoinStarted),
		errors.Is(err, domain.ErrRoleAlreadyTaken),
		errors.Is(err, domain.ErrRoomNotReady),
		errors.Is(err, domain.ErrGameAlreadyStarted):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
