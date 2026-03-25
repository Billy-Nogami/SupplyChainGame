package memory

import (
	"context"
	"sync"

	"supply-chain-simulator/internal/domain"
)

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]domain.GameSession
	byRoomID map[string]string
}

func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]domain.GameSession),
		byRoomID: make(map[string]string),
	}
}

func (s *SessionStore) Save(_ context.Context, session domain.GameSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[session.ID] = session
	s.byRoomID[session.RoomID] = session.ID

	return nil
}

func (s *SessionStore) GetByID(_ context.Context, sessionID string) (domain.GameSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return domain.GameSession{}, domain.ErrSessionNotFound
	}

	return session, nil
}

func (s *SessionStore) GetByRoomID(_ context.Context, roomID string) (domain.GameSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessionID, ok := s.byRoomID[roomID]
	if !ok {
		return domain.GameSession{}, domain.ErrSessionNotFound
	}

	session, ok := s.sessions[sessionID]
	if !ok {
		return domain.GameSession{}, domain.ErrSessionNotFound
	}

	return session, nil
}
