package usecase

import (
	"context"
	"errors"

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
	roomStore     RoomStore
	sessionStore  SessionStore
	decisionStore DecisionStore
	scenarios     ScenarioRepository
	exporter      SessionExporter
	idGenerator   IDGenerator
	clock         Clock
}

func NewGameService(
	roomStore RoomStore,
	sessionStore SessionStore,
	decisionStore DecisionStore,
	scenarios ScenarioRepository,
	exporter SessionExporter,
	idGenerator IDGenerator,
	clock Clock,
) *GameService {
	return &GameService{
		roomStore:     roomStore,
		sessionStore:  sessionStore,
		decisionStore: decisionStore,
		scenarios:     scenarios,
		exporter:      exporter,
		idGenerator:   idGenerator,
		clock:         clock,
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

	room.CurrentWeek = currentPlayableWeek(session)
	room.ScenarioID = scenario.ID

	if err := s.roomStore.Save(ctx, room); err != nil {
		return domain.GameSession{}, err
	}
	if err := s.sessionStore.Save(ctx, session); err != nil {
		return domain.GameSession{}, err
	}

	return session, nil
}

func (s *GameService) SubmitOrder(ctx context.Context, roomID, playerID string, order int) (WeeklyDecisionsSnapshot, error) {
	if order < 0 {
		return WeeklyDecisionsSnapshot{}, domain.ErrNegativeDecision
	}

	room, err := s.roomStore.GetByID(ctx, roomID)
	if err != nil {
		return WeeklyDecisionsSnapshot{}, err
	}

	role, err := room.RoleOfPlayer(playerID)
	if err != nil {
		return WeeklyDecisionsSnapshot{}, err
	}

	session, err := s.sessionStore.GetByRoomID(ctx, roomID)
	if err != nil {
		return WeeklyDecisionsSnapshot{}, err
	}
	if session.Status != domain.GameStatusActive {
		return WeeklyDecisionsSnapshot{}, domain.ErrSessionNotActive
	}

	week := currentPlayableWeek(session)
	decisions, err := s.decisionStore.GetByRoomAndWeek(ctx, roomID, week)
	if err != nil {
		if !errors.Is(err, domain.ErrWeekDecisionsNotFound) {
			return WeeklyDecisionsSnapshot{}, err
		}
		decisions = WeeklyDecisions{
			RoomID: roomID,
			Week:   week,
			Orders: make(map[domain.Role]int, len(domain.AllRoles)),
		}
	}

	decisions.Orders[role] = order
	decisions.UpdatedAt = s.clock.Now()
	if err := s.decisionStore.Save(ctx, decisions); err != nil {
		return WeeklyDecisionsSnapshot{}, err
	}

	return decisions.Snapshot(), nil
}

func (s *GameService) AdvanceWeek(ctx context.Context, roomID string) (domain.WeekState, error) {
	room, err := s.roomStore.GetByID(ctx, roomID)
	if err != nil {
		return domain.WeekState{}, err
	}

	session, err := s.sessionStore.GetByRoomID(ctx, roomID)
	if err != nil {
		return domain.WeekState{}, err
	}

	week := currentPlayableWeek(session)
	decisions, err := s.decisionStore.GetByRoomAndWeek(ctx, roomID, week)
	if err != nil {
		if errors.Is(err, domain.ErrWeekDecisionsNotFound) {
			return domain.WeekState{}, domain.ErrWeekNotReady
		}
		return domain.WeekState{}, err
	}
	if len(decisions.Orders) != len(domain.AllRoles) {
		return domain.WeekState{}, domain.ErrWeekNotReady
	}

	weekState, err := session.AdvanceWeek(decisions.Orders, s.clock.Now())
	if err != nil {
		return domain.WeekState{}, err
	}

	room.CurrentWeek = currentPlayableWeek(session)
	room.Status = session.Status
	room.UpdatedAt = s.clock.Now()

	if err := s.sessionStore.Save(ctx, session); err != nil {
		return domain.WeekState{}, err
	}
	if err := s.roomStore.Save(ctx, room); err != nil {
		return domain.WeekState{}, err
	}
	if err := s.decisionStore.DeleteByRoomAndWeek(ctx, roomID, week); err != nil {
		return domain.WeekState{}, err
	}

	return weekState, nil
}

func (s *GameService) GetSession(ctx context.Context, sessionID string) (domain.GameSession, error) {
	return s.sessionStore.GetByID(ctx, sessionID)
}

func (s *GameService) GetSessionByRoom(ctx context.Context, roomID string) (domain.GameSession, error) {
	return s.sessionStore.GetByRoomID(ctx, roomID)
}

func (s *GameService) GetWeeks(ctx context.Context, roomID string) ([]domain.WeekState, error) {
	session, err := s.sessionStore.GetByRoomID(ctx, roomID)
	if err != nil {
		return nil, err
	}

	weeks := make([]domain.WeekState, len(session.History))
	copy(weeks, session.History)

	return weeks, nil
}

func (s *GameService) GetAnalytics(ctx context.Context, roomID string) (domain.SessionAnalytics, error) {
	session, err := s.sessionStore.GetByRoomID(ctx, roomID)
	if err != nil {
		return domain.SessionAnalytics{}, err
	}

	return domain.CalculateSessionAnalytics(session), nil
}

func (s *GameService) ExportSession(ctx context.Context, roomID string) (ExportedFile, error) {
	session, err := s.sessionStore.GetByRoomID(ctx, roomID)
	if err != nil {
		return ExportedFile{}, err
	}

	analytics := domain.CalculateSessionAnalytics(session)

	return s.exporter.ExportSession(ctx, session, analytics)
}

func (s *GameService) GetPendingDecisions(ctx context.Context, roomID string) (WeeklyDecisionsSnapshot, error) {
	session, err := s.sessionStore.GetByRoomID(ctx, roomID)
	if err != nil {
		return WeeklyDecisionsSnapshot{}, err
	}

	week := currentPlayableWeek(session)
	decisions, err := s.decisionStore.GetByRoomAndWeek(ctx, roomID, week)
	if err != nil {
		if errors.Is(err, domain.ErrWeekDecisionsNotFound) {
			return WeeklyDecisions{
				RoomID: roomID,
				Week:   week,
				Orders: map[domain.Role]int{},
			}.Snapshot(), nil
		}
		return WeeklyDecisionsSnapshot{}, err
	}

	return decisions.Snapshot(), nil
}

func (s *GameService) resolveScenario(ctx context.Context, scenarioID string) (domain.Scenario, error) {
	if scenarioID == "" {
		return s.scenarios.GetDefault(ctx)
	}

	return s.scenarios.GetByID(ctx, scenarioID)
}

func currentPlayableWeek(session domain.GameSession) int {
	if session.Status == domain.GameStatusFinished {
		return session.CurrentWeek
	}
	return session.CurrentWeek + 1
}
