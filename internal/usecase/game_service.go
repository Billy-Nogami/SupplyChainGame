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

type ScenarioRepository interface {
	GetDefault(ctx context.Context) (domain.Scenario, error)
	GetByID(ctx context.Context, scenarioID string) (domain.Scenario, error)
}

type GameService struct {
	roomStore    RoomStore
	sessionStore SessionStore
	scenarios    ScenarioRepository
	idGenerator  IDGenerator
	clock        Clock
}

func NewGameService(
	roomStore RoomStore,
	sessionStore SessionStore,
	scenarios ScenarioRepository,
	idGenerator IDGenerator,
	clock Clock,
) *GameService {
	return &GameService{
		roomStore:    roomStore,
		sessionStore: sessionStore,
		scenarios:    scenarios,
		idGenerator:  idGenerator,
		clock:        clock,
	}
}

func (s *GameService) StartGame(ctx context.Context, roomID, scenarioID string) (domain.GameSession, error) {
	room, err := s.roomStore.GetByID(ctx, roomID)
	if err != nil {
		return domain.GameSession{}, err
	}

	scenario, err := s.resolveScenario(ctx, scenarioID)
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

func (s *GameService) resolveScenario(ctx context.Context, scenarioID string) (domain.Scenario, error) {
	if scenarioID == "" {
		return s.scenarios.GetDefault(ctx)
	}

	return s.scenarios.GetByID(ctx, scenarioID)
}
