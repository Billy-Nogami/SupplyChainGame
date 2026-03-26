package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"supply-chain-simulator/internal/domain"
	"supply-chain-simulator/internal/usecase"
)

type RoomEventSubscriber interface {
	Subscribe(roomID string) (<-chan usecase.RoomEvent, func())
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request, roomID string) {
	if s.events == nil {
		writeError(w, http.StatusServiceUnavailable, "room events are not configured")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming is not supported")
		return
	}

	initialEvent, err := s.snapshotEvent(r.Context(), roomID)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	writeSSEEvent(w, initialEvent)
	flusher.Flush()

	events, cancel := s.events.Subscribe(roomID)
	defer cancel()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			writeSSEEvent(w, event)
			flusher.Flush()
		}
	}
}

func (s *Server) snapshotEvent(ctx context.Context, roomID string) (usecase.RoomEvent, error) {
	room, err := s.roomService.GetRoom(ctx, roomID)
	if err != nil {
		return usecase.RoomEvent{}, err
	}

	event := usecase.RoomEvent{
		Type:   "room.snapshot",
		RoomID: roomID,
		Room:   &room,
	}

	session, err := s.gameService.GetSessionByRoom(ctx, roomID)
	if err == nil {
		analytics, analyticsErr := s.gameService.GetAnalytics(ctx, roomID)
		if analyticsErr != nil {
			return usecase.RoomEvent{}, analyticsErr
		}
		decisions, decisionsErr := s.gameService.GetPendingDecisions(ctx, roomID)
		if decisionsErr != nil {
			return usecase.RoomEvent{}, decisionsErr
		}

		event.Session = &session
		event.Analytics = &analytics
		event.Decisions = &decisions
	} else if err != nil && !errors.Is(err, domain.ErrSessionNotFound) {
		return usecase.RoomEvent{}, err
	}

	return event, nil
}

func writeSSEEvent(w http.ResponseWriter, event usecase.RoomEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}

	_, _ = fmt.Fprintf(w, "event: %s\n", event.Type)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", payload)
}
