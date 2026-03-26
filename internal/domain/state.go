package domain

type NodeState struct {
	ID             string `json:"id"`
	Role           Role   `json:"role"`
	Inventory      int    `json:"inventory"`
	Backlog        int    `json:"backlog"`
	IncomingOrder  int    `json:"incoming_order"`
	IncomingGoods  int    `json:"incoming_goods"`
	PlacedOrder    int    `json:"placed_order"`
	ActualShipment int    `json:"actual_shipment"`
	WeeklyCost     int    `json:"weekly_cost"`
}

type Pipeline struct {
	GoodsPipeline []int `json:"goods_pipeline"`
	OrderPipeline []int `json:"order_pipeline"`
}

type WeekState struct {
	Week      int         `json:"week"`
	Nodes     []NodeState `json:"nodes"`
	TotalCost int         `json:"total_cost"`
}
