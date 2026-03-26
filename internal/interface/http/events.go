package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"supply-chain-simulator/internal/usecase"
)

type RoomEventSubscriber interface {
	Subscribe(roomID string) (<-chan usecase.RoomEvent, func())
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request, roomID string) {
	playerID := r.URL.Query().Get("player_id")
	if playerID == "" {
		writeError(w, http.StatusBadRequest, "player_id is required")
		return
	}
	if s.events == nil {
		writeError(w, http.StatusServiceUnavailable, "room events are not configured")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming is not supported")
		return
	}

	initialEvent, err := s.snapshotEvent(r.Context(), roomID, playerID)
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
			playerState, err := s.gameService.GetPlayerState(context.Background(), roomID, playerID)
			if err != nil {
				continue
			}
			writeSSEEvent(w, usecase.PlayerRoomEvent{
				Type:  event.Type,
				State: playerState,
			})
			flusher.Flush()
		}
	}
}

func (s *Server) snapshotEvent(ctx context.Context, roomID, playerID string) (usecase.PlayerRoomEvent, error) {
	_, err := s.roomService.GetRoom(ctx, roomID)
	if err != nil {
		return usecase.PlayerRoomEvent{}, err
	}

	state, err := s.gameService.GetPlayerState(ctx, roomID, playerID)
	if err != nil {
		return usecase.PlayerRoomEvent{}, err
	}

	return usecase.PlayerRoomEvent{
		Type:  "room.snapshot",
		State: state,
	}, nil
}

func writeSSEEvent(w http.ResponseWriter, event usecase.PlayerRoomEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}

	_, _ = fmt.Fprintf(w, "event: %s\n", event.Type)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", payload)
}
