package usecase

import (
	"context"
	"errors"
	"log"

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
	roomStore      RoomStore
	sessionStore   SessionStore
	decisionStore  DecisionStore
	scenarios      ScenarioRepository
	exporter       SessionExporter
	eventPublisher RoomEventPublisher
	idGenerator    IDGenerator
	clock          Clock
}

func NewGameService(
	roomStore RoomStore,
	sessionStore SessionStore,
	decisionStore DecisionStore,
	scenarios ScenarioRepository,
	exporter SessionExporter,
	eventPublisher RoomEventPublisher,
	idGenerator IDGenerator,
	clock Clock,
) *GameService {
	if eventPublisher == nil {
		eventPublisher = NopRoomEventPublisher{}
	}

	return &GameService{
		roomStore:      roomStore,
		sessionStore:   sessionStore,
		decisionStore:  decisionStore,
		scenarios:      scenarios,
		exporter:       exporter,
		eventPublisher: eventPublisher,
		idGenerator:    idGenerator,
		clock:          clock,
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
	scenario, err = scenario.MaterializeDemand(room.MaxWeeks, now.UnixNano())
	if err != nil {
		return domain.GameSession{}, err
	}
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
	s.publishRoomEvent(ctx, "game.started", room, session, nil)
	log.Printf("game_started room_id=%s session_id=%s scenario_id=%s max_weeks=%d", room.ID, session.ID, scenario.ID, session.MaxWeeks)

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

	snapshot := decisions.Snapshot()
	s.publishRoomEvent(ctx, "game.order_submitted", room, session, &snapshot)
	log.Printf(
		"game_order_submitted room_id=%s session_id=%s player_id=%s role=%s week=%d order=%d submitted=%d expected=%d",
		room.ID,
		session.ID,
		playerID,
		role,
		week,
		order,
		len(snapshot.SubmittedRoles),
		len(domain.AllRoles),
	)

	return snapshot, nil
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
	s.publishRoomEvent(ctx, "game.week_advanced", room, session, nil)
	log.Printf(
		"game_week_advanced room_id=%s session_id=%s completed_week=%d next_week=%d status=%s total_cost=%d",
		room.ID,
		session.ID,
		weekState.Week,
		currentPlayableWeek(session),
		session.Status,
		weekState.TotalCost,
	)

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
	log.Printf("game_export_requested room_id=%s session_id=%s weeks=%d total_cost=%d", roomID, session.ID, analytics.TotalWeeks, analytics.TotalCost)

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

func (s *GameService) GetPlayerState(ctx context.Context, roomID, playerID string) (PlayerGameState, error) {
	room, err := s.roomStore.GetByID(ctx, roomID)
	if err != nil {
		return PlayerGameState{}, err
	}

	player, err := findPlayer(room, playerID)
	if err != nil {
		return PlayerGameState{}, err
	}

	state := PlayerGameState{
		RoomID:      room.ID,
		PlayerID:    player.ID,
		PlayerName:  player.Name,
		Role:        player.Role,
		RoomStatus:  room.Status,
		CurrentWeek: room.CurrentWeek,
		MaxWeeks:    room.MaxWeeks,
		ScenarioID:  room.ScenarioID,
		Players:     toPlayerSummaries(room),
		OwnHistory:  []domain.NodeState{},
	}

	if room.Status == domain.GameStatusWaiting {
		return state, nil
	}

	session, err := s.sessionStore.GetByRoomID(ctx, roomID)
	if err != nil {
		return PlayerGameState{}, err
	}

	decisions, err := s.GetPendingDecisions(ctx, roomID)
	if err != nil {
		return PlayerGameState{}, err
	}

	analytics := domain.CalculateSessionAnalytics(session)
	ownNode := nodeByRole(session.Nodes, player.Role)
	ownHistory := make([]domain.NodeState, 0, len(session.History))
	for _, week := range session.History {
		ownHistory = append(ownHistory, nodeByRole(week.Nodes, player.Role))
	}

	state.CurrentWeek = currentPlayableWeek(session)
	state.ScenarioID = session.Scenario.ID
	state.OrdersSubmitted = len(decisions.SubmittedRoles)
	state.OrdersExpected = len(domain.AllRoles)
	state.WeekReady = decisions.Ready
	state.OwnNode = &ownNode
	state.OwnHistory = ownHistory
	state.TotalSystemCost = analytics.TotalCost
	if ownOrder, ok := decisions.Orders[player.Role]; ok {
		state.OwnOrderSubmitted = true
		state.OwnCurrentOrder = ownOrder
	}
	if ownAnalytics, ok := playerAnalyticsByRole(analytics, player.Role); ok {
		state.OwnAnalytics = &ownAnalytics
	}

	return state, nil
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

func findPlayer(room domain.Room, playerID string) (domain.Player, error) {
	for _, player := range room.Players {
		if player.ID == playerID {
			return player, nil
		}
	}

	return domain.Player{}, domain.ErrPlayerNotFound
}

func toPlayerSummaries(room domain.Room) []PlayerSummary {
	players := make([]PlayerSummary, 0, len(room.Players))
	for _, player := range room.Players {
		players = append(players, PlayerSummary{
			Name:      player.Name,
			Role:      player.Role,
			Connected: player.Connected,
		})
	}

	return players
}

func playerAnalyticsByRole(analytics domain.SessionAnalytics, role domain.Role) (PlayerAnalytics, bool) {
	for _, node := range analytics.NodeAnalytics {
		if node.Role != role {
			continue
		}

		return PlayerAnalytics{
			Role:             node.Role,
			TotalCost:        node.TotalCost,
			AverageInventory: node.AverageInventory,
			TotalBacklog:     node.TotalBacklog,
			TotalOrders:      node.TotalOrders,
			OrderVariance:    node.OrderVariance,
		}, true
	}

	return PlayerAnalytics{}, false
}

func nodeByRole(nodes []domain.NodeState, role domain.Role) domain.NodeState {
	for _, node := range nodes {
		if node.Role == role {
			return node
		}
	}

	return domain.NodeState{Role: role}
}

func (s *GameService) publishRoomEvent(
	ctx context.Context,
	eventType string,
	room domain.Room,
	session domain.GameSession,
	decisions *WeeklyDecisionsSnapshot,
) {
	analytics := domain.CalculateSessionAnalytics(session)
	_ = s.eventPublisher.PublishRoomEvent(ctx, RoomEvent{
		Type:       eventType,
		RoomID:     room.ID,
		OccurredAt: s.clock.Now(),
		Room:       &room,
		Session:    &session,
		Decisions:  decisions,
		Analytics:  &analytics,
	})
}
