package usecase

import (
	"context"

	"supply-chain-simulator/internal/domain"
)

type SessionStore interface {
	Save(ctx context.Context, session domain.GameSession) error
	GetByID(ctx context.Context, sessionID string) (domain.GameSession, error)
	GetByRoomID(ctx context.Context, roomID string) (domain.GameSession, error)
}

type GameService struct {
	roomStore    RoomStore
	sessionStore SessionStore
	idGenerator  IDGenerator
	clock        Clock
}

func NewGameService(roomStore RoomStore, sessionStore SessionStore, idGenerator IDGenerator, clock Clock) *GameService {
	return &GameService{
		roomStore:    roomStore,
		sessionStore: sessionStore,
		idGenerator:  idGenerator,
		clock:        clock,
	}
}

func (s *GameService) StartGame(ctx context.Context, roomID string, scenario domain.Scenario) (domain.GameSession, error) {
	room, err := s.roomStore.GetByID(ctx, roomID)
	if err != nil {
		return domain.GameSession{}, err
	}

	now := s.clock.Now()
	if err := room.Start(now); err != nil {
		return domain.GameSession{}, err
	}

	sessionID, err := s.idGenerator.NewID()
	if err != nil {
		return domain.GameSession{}, err
	}

	session, err := domain.NewGameSession(sessionID, room.ID, scenario, room.MaxWeeks, now)
	if err != nil {
		return domain.GameSession{}, err
	}

	room.CurrentWeek = session.CurrentWeek
	room.ScenarioID = scenario.ID
	if err := s.roomStore.Save(ctx, room); err != nil {
		return domain.GameSession{}, err
	}
	if err := s.sessionStore.Save(ctx, session); err != nil {
		return domain.GameSession{}, err
	}

	return session, nil
}

func (s *GameService) GetSession(ctx context.Context, sessionID string) (domain.GameSession, error) {
	return s.sessionStore.GetByID(ctx, sessionID)
}

func (s *GameService) GetSessionByRoom(ctx context.Context, roomID string) (domain.GameSession, error) {
	return s.sessionStore.GetByRoomID(ctx, roomID)
}
