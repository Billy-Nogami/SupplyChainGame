package domain

type NodeState struct {
	ID             string
	Role           Role
	Inventory      int
	Backlog        int
	IncomingOrder  int
	IncomingGoods  int
	PlacedOrder    int
	ActualShipment int
	WeeklyCost     int
}

type Pipeline struct {
	GoodsPipeline []int
	OrderPipeline []int
}

type WeekState struct {
	Week      int
	Nodes     []NodeState
	TotalCost int
}
