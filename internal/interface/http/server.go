package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"supply-chain-simulator/internal/domain"
	"supply-chain-simulator/internal/usecase"
)

type RoomService interface {
	CreateRoom(ctx context.Context, maxWeeks int) (domain.Room, error)
	GetRoom(ctx context.Context, roomID string) (domain.Room, error)
	JoinRoom(ctx context.Context, roomID, playerName string) (domain.Room, error)
	AssignRole(ctx context.Context, roomID, playerID string, role domain.Role) (domain.Room, error)
}

type GameService interface {
	StartGame(ctx context.Context, roomID, scenarioID string) (domain.GameSession, error)
	SubmitOrder(ctx context.Context, roomID, playerID string, order int) (usecase.WeeklyDecisionsSnapshot, error)
	AdvanceWeek(ctx context.Context, roomID string) (domain.WeekState, error)
	GetSessionByRoom(ctx context.Context, roomID string) (domain.GameSession, error)
	GetWeeks(ctx context.Context, roomID string) ([]domain.WeekState, error)
	GetAnalytics(ctx context.Context, roomID string) (domain.SessionAnalytics, error)
	GetPendingDecisions(ctx context.Context, roomID string) (usecase.WeeklyDecisionsSnapshot, error)
	ExportSession(ctx context.Context, roomID string) (usecase.ExportedFile, error)
}

type Server struct {
	roomService RoomService
	gameService GameService
	events      RoomEventSubscriber
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

type submitOrderRequest struct {
	PlayerID string `json:"player_id"`
	Order    int    `json:"order"`
}

func NewServer(roomService RoomService, gameService GameService, events RoomEventSubscriber) *Server {
	server := &Server{
		roomService: roomService,
		gameService: gameService,
		events:      events,
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
	case "weeks":
		weeks, err := s.gameService.GetWeeks(r.Context(), roomID)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, weeks)
	case "analytics":
		analytics, err := s.gameService.GetAnalytics(r.Context(), roomID)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, analytics)
	case "decisions":
		decisions, err := s.gameService.GetPendingDecisions(r.Context(), roomID)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, decisions)
	case "export":
		exportedFile, err := s.gameService.ExportSession(r.Context(), roomID)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeFile(w, http.StatusOK, exportedFile)
	case "events":
		s.handleEvents(w, r, roomID)
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
	case "orders":
		s.handleSubmitOrder(w, r, roomID)
	case "next":
		s.handleAdvanceWeek(w, r, roomID)
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

func (s *Server) handleSubmitOrder(w http.ResponseWriter, r *http.Request, roomID string) {
	var req submitOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	decisions, err := s.gameService.SubmitOrder(r.Context(), roomID, req.PlayerID, req.Order)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, decisions)
}

func (s *Server) handleAdvanceWeek(w http.ResponseWriter, r *http.Request, roomID string) {
	weekState, err := s.gameService.AdvanceWeek(r.Context(), roomID)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, weekState)
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
		errors.Is(err, domain.ErrSessionNotFound),
		errors.Is(err, domain.ErrWeekDecisionsNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrInvalidMaxWeeks),
		errors.Is(err, domain.ErrEmptyPlayerName),
		errors.Is(err, domain.ErrPlayerAlreadyIn),
		errors.Is(err, domain.ErrPlayerNotFound),
		errors.Is(err, domain.ErrInvalidRole),
		errors.Is(err, domain.ErrNegativeDecision),
		errors.Is(err, domain.ErrPlayerRoleMissing):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrRoomFull),
		errors.Is(err, domain.ErrCannotJoinStarted),
		errors.Is(err, domain.ErrRoleAlreadyTaken),
		errors.Is(err, domain.ErrRoomNotReady),
		errors.Is(err, domain.ErrGameAlreadyStarted),
		errors.Is(err, domain.ErrWeekNotReady),
		errors.Is(err, domain.ErrSessionNotActive),
		errors.Is(err, domain.ErrWeekLimitReached):
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

func writeFile(w http.ResponseWriter, status int, file usecase.ExportedFile) {
	w.Header().Set("Content-Type", file.ContentType)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+file.FileName+"\"")
	w.Header().Set("Content-Length", strconv.Itoa(len(file.Content)))
	w.WriteHeader(status)
	_, _ = w.Write(file.Content)
}
