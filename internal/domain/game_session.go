package domain

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrScenarioDemandEmpty = errors.New("scenario consumer demand must not be empty")
	ErrInvalidDelay        = errors.New("scenario delays must be positive")
	ErrNegativeValue       = errors.New("scenario values must not be negative")
	ErrSessionNotActive    = errors.New("game session is not active")
	ErrSessionNotFound     = errors.New("game session not found")
	ErrWeekLimitReached    = errors.New("game session reached max weeks")
	ErrMissingDecision     = errors.New("missing decision for role")
	ErrNegativeDecision    = errors.New("decision must not be negative")
)

type GameSession struct {
	ID              string
	RoomID          string
	Status          GameStatus
	CurrentWeek     int
	MaxWeeks        int
	Scenario        Scenario
	Nodes           []NodeState
	GoodsPipelines  map[Role][]int
	OrderPipelines  map[Role][]int
	ProductionQueue []int
	History         []WeekState
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func NewGameSession(id, roomID string, scenario Scenario, maxWeeks int, now time.Time) (GameSession, error) {
	if maxWeeks <= 0 {
		return GameSession{}, ErrInvalidMaxWeeks
	}
	if err := scenario.Validate(); err != nil {
		return GameSession{}, err
	}

	nodes := make([]NodeState, 0, len(AllRoles))
	for _, role := range AllRoles {
		nodes = append(nodes, NodeState{
			ID:        fmt.Sprintf("%s-%s", id, role),
			Role:      role,
			Inventory: scenario.InitialInventory,
			Backlog:   scenario.InitialBacklog,
		})
	}

	session := GameSession{
		ID:              id,
		RoomID:          roomID,
		Status:          GameStatusActive,
		CurrentWeek:     0,
		MaxWeeks:        maxWeeks,
		Scenario:        scenario,
		Nodes:           nodes,
		GoodsPipelines:  make(map[Role][]int, 3),
		OrderPipelines:  make(map[Role][]int, 3),
		ProductionQueue: copyInts(scenario.InitialPipelineGoods),
		History:         make([]WeekState, 0, maxWeeks),
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	session.GoodsPipelines[RoleDistributor] = copyInts(scenario.InitialPipelineGoods)
	session.GoodsPipelines[RoleWholesaler] = copyInts(scenario.InitialPipelineGoods)
	session.GoodsPipelines[RoleRetailer] = copyInts(scenario.InitialPipelineGoods)

	session.OrderPipelines[RoleFactory] = copyInts(scenario.InitialPipelineOrders)
	session.OrderPipelines[RoleDistributor] = copyInts(scenario.InitialPipelineOrders)
	session.OrderPipelines[RoleWholesaler] = copyInts(scenario.InitialPipelineOrders)

	return session, nil
}

func (s *GameSession) AdvanceWeek(decisions map[Role]int, now time.Time) (WeekState, error) {
	if s.Status != GameStatusActive {
		return WeekState{}, ErrSessionNotActive
	}
	if s.CurrentWeek >= s.MaxWeeks {
		return WeekState{}, ErrWeekLimitReached
	}

	for _, role := range AllRoles {
		order, ok := decisions[role]
		if !ok {
			return WeekState{}, fmt.Errorf("%w: %s", ErrMissingDecision, role)
		}
		if order < 0 {
			return WeekState{}, fmt.Errorf("%w: %s", ErrNegativeDecision, role)
		}
	}

	nextWeek := s.CurrentWeek + 1
	incomingGoods := map[Role]int{
		RoleDistributor: shiftPipeline(s.GoodsPipelines, RoleDistributor),
		RoleWholesaler:  shiftPipeline(s.GoodsPipelines, RoleWholesaler),
		RoleRetailer:    shiftPipeline(s.GoodsPipelines, RoleRetailer),
		RoleFactory:     shiftSlice(&s.ProductionQueue),
	}
	incomingOrders := map[Role]int{
		RoleRetailer:    s.Scenario.DemandForWeek(nextWeek),
		RoleWholesaler:  shiftPipeline(s.OrderPipelines, RoleWholesaler),
		RoleDistributor: shiftPipeline(s.OrderPipelines, RoleDistributor),
		RoleFactory:     shiftPipeline(s.OrderPipelines, RoleFactory),
	}

	totalCost := 0
	nextNodes := make([]NodeState, 0, len(s.Nodes))
	for _, role := range AllRoles {
		prev := s.mustNode(role)
		available := prev.Inventory + incomingGoods[role]
		required := incomingOrders[role] + prev.Backlog
		shipment := minInt(available, required)
		backlog := maxInt(required-shipment, 0)
		inventory := available - shipment
		cost := inventory*s.Scenario.HoldingCost + backlog*s.Scenario.BacklogCost

		next := NodeState{
			ID:             prev.ID,
			Role:           role,
			Inventory:      inventory,
			Backlog:        backlog,
			IncomingOrder:  incomingOrders[role],
			IncomingGoods:  incomingGoods[role],
			PlacedOrder:    decisions[role],
			ActualShipment: shipment,
			WeeklyCost:     cost,
		}
		nextNodes = append(nextNodes, next)
		totalCost += cost
	}

	appendPipeline(s.OrderPipelines, RoleWholesaler, decisions[RoleRetailer])
	appendPipeline(s.OrderPipelines, RoleDistributor, decisions[RoleWholesaler])
	appendPipeline(s.OrderPipelines, RoleFactory, decisions[RoleDistributor])
	s.ProductionQueue = append(s.ProductionQueue, decisions[RoleFactory])

	appendPipeline(s.GoodsPipelines, RoleDistributor, nodeByRole(nextNodes, RoleFactory).ActualShipment)
	appendPipeline(s.GoodsPipelines, RoleWholesaler, nodeByRole(nextNodes, RoleDistributor).ActualShipment)
	appendPipeline(s.GoodsPipelines, RoleRetailer, nodeByRole(nextNodes, RoleWholesaler).ActualShipment)

	weekState := WeekState{
		Week:      nextWeek,
		Nodes:     nextNodes,
		TotalCost: totalCost,
	}

	s.Nodes = nextNodes
	s.CurrentWeek = nextWeek
	s.History = append(s.History, weekState)
	s.UpdatedAt = now
	if s.CurrentWeek >= s.MaxWeeks {
		s.Status = GameStatusFinished
	}

	return weekState, nil
}

func (s GameSession) mustNode(role Role) NodeState {
	return nodeByRole(s.Nodes, role)
}

func nodeByRole(nodes []NodeState, role Role) NodeState {
	for _, node := range nodes {
		if node.Role == role {
			return node
		}
	}

	return NodeState{Role: role}
}

func shiftPipeline(pipelines map[Role][]int, role Role) int {
	queue := pipelines[role]
	if len(queue) == 0 {
		return 0
	}

	incoming := queue[0]
	pipelines[role] = queue[1:]

	return incoming
}

func appendPipeline(pipelines map[Role][]int, role Role, value int) {
	pipelines[role] = append(pipelines[role], value)
}

func shiftSlice(queue *[]int) int {
	if len(*queue) == 0 {
		return 0
	}

	incoming := (*queue)[0]
	*queue = (*queue)[1:]

	return incoming
}

func copyInts(values []int) []int {
	if len(values) == 0 {
		return nil
	}

	result := make([]int, len(values))
	copy(result, values)

	return result
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
