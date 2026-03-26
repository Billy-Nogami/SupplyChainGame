package usecase

import "supply-chain-simulator/internal/domain"

type PlayerSummary struct {
	Name      string      `json:"name"`
	Role      domain.Role `json:"role"`
	Connected bool        `json:"connected"`
}

type PlayerAnalytics struct {
	Role             domain.Role `json:"role"`
	TotalCost        int         `json:"total_cost"`
	AverageInventory float64     `json:"average_inventory"`
	TotalBacklog     int         `json:"total_backlog"`
	TotalOrders      int         `json:"total_orders"`
	OrderVariance    float64     `json:"order_variance"`
}

type PlayerGameState struct {
	RoomID            string             `json:"room_id"`
	PlayerID          string             `json:"player_id"`
	PlayerName        string             `json:"player_name"`
	Role              domain.Role        `json:"role"`
	RoomStatus        domain.GameStatus  `json:"room_status"`
	CurrentWeek       int                `json:"current_week"`
	MaxWeeks          int                `json:"max_weeks"`
	ScenarioID        string             `json:"scenario_id"`
	Players           []PlayerSummary    `json:"players"`
	OrdersSubmitted   int                `json:"orders_submitted"`
	OrdersExpected    int                `json:"orders_expected"`
	WeekReady         bool               `json:"week_ready"`
	OwnOrderSubmitted bool               `json:"own_order_submitted"`
	OwnCurrentOrder   int                `json:"own_current_order"`
	OwnNode           *domain.NodeState  `json:"own_node,omitempty"`
	OwnHistory        []domain.NodeState `json:"own_history"`
	TotalSystemCost   int                `json:"total_system_cost"`
	OwnAnalytics      *PlayerAnalytics   `json:"own_analytics,omitempty"`
}

type PlayerRoomEvent struct {
	Type  string          `json:"type"`
	State PlayerGameState `json:"state"`
}
