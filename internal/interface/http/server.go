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
}

type Server struct {
	roomService RoomService
	mux         *http.ServeMux
}

type createRoomRequest struct {
	MaxWeeks int `json:"max_weeks"`
}

type joinRoomRequest struct {
	Name string `json:"name"`
}

func NewServer(roomService RoomService) *Server {
	server := &Server{
		roomService: roomService,
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
	s.mux.HandleFunc("GET /rooms/", s.handleGetRoom)
	s.mux.HandleFunc("POST /rooms/", s.handleJoinRoom)
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

func (s *Server) handleGetRoom(w http.ResponseWriter, r *http.Request) {
	roomID, action := parseRoomPath(r.URL.Path)
	if roomID == "" || action != "" {
		http.NotFound(w, r)
		return
	}

	room, err := s.roomService.GetRoom(r.Context(), roomID)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, room)
}

func (s *Server) handleJoinRoom(w http.ResponseWriter, r *http.Request) {
	roomID, action := parseRoomPath(r.URL.Path)
	if roomID == "" || action != "players" {
		http.NotFound(w, r)
		return
	}

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
	case errors.Is(err, domain.ErrRoomNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrInvalidMaxWeeks),
		errors.Is(err, domain.ErrEmptyPlayerName),
		errors.Is(err, domain.ErrPlayerAlreadyIn):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrRoomFull),
		errors.Is(err, domain.ErrCannotJoinStarted):
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
