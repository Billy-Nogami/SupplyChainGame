package domain

type NodeAnalytics struct {
	Role             Role    `json:"role"`
	TotalCost        int     `json:"total_cost"`
	AverageInventory float64 `json:"average_inventory"`
	TotalBacklog     int     `json:"total_backlog"`
	TotalOrders      int     `json:"total_orders"`
	OrderVariance    float64 `json:"order_variance"`
}

type SessionAnalytics struct {
	TotalWeeks     int              `json:"total_weeks"`
	TotalCost      int              `json:"total_cost"`
	DemandVariance float64          `json:"demand_variance"`
	NodeAnalytics  []NodeAnalytics  `json:"node_analytics"`
	OrderVariance  map[Role]float64 `json:"order_variance"`
	BullwhipEffect map[Role]float64 `json:"bullwhip_effect"`
}

func CalculateSessionAnalytics(session GameSession) SessionAnalytics {
	result := SessionAnalytics{
		TotalWeeks:     len(session.History),
		NodeAnalytics:  make([]NodeAnalytics, 0, len(AllRoles)),
		OrderVariance:  make(map[Role]float64, len(AllRoles)),
		BullwhipEffect: make(map[Role]float64, len(AllRoles)),
	}
	if len(session.History) == 0 {
		return result
	}

	demandSeries := make([]int, 0, len(session.History))
	perRoleOrders := make(map[Role][]int, len(AllRoles))
	perRoleInventory := make(map[Role][]int, len(AllRoles))
	perRoleBacklog := make(map[Role][]int, len(AllRoles))
	perRoleCost := make(map[Role]int, len(AllRoles))

	for _, week := range session.History {
		result.TotalCost += week.TotalCost
		for _, node := range week.Nodes {
			perRoleOrders[node.Role] = append(perRoleOrders[node.Role], node.PlacedOrder)
			perRoleInventory[node.Role] = append(perRoleInventory[node.Role], node.Inventory)
			perRoleBacklog[node.Role] = append(perRoleBacklog[node.Role], node.Backlog)
			perRoleCost[node.Role] += node.WeeklyCost
			if node.Role == RoleRetailer {
				demandSeries = append(demandSeries, node.IncomingOrder)
			}
		}
	}

	result.DemandVariance = varianceInts(demandSeries)
	for _, role := range AllRoles {
		orders := perRoleOrders[role]
		inventory := perRoleInventory[role]
		backlog := perRoleBacklog[role]
		roleVariance := varianceInts(orders)
		result.OrderVariance[role] = roleVariance
		if result.DemandVariance > 0 {
			result.BullwhipEffect[role] = roleVariance / result.DemandVariance
		} else {
			result.BullwhipEffect[role] = 0
		}

		result.NodeAnalytics = append(result.NodeAnalytics, NodeAnalytics{
			Role:             role,
			TotalCost:        perRoleCost[role],
			AverageInventory: averageInts(inventory),
			TotalBacklog:     sumInts(backlog),
			TotalOrders:      sumInts(orders),
			OrderVariance:    roleVariance,
		})
	}

	return result
}

func averageInts(values []int) float64 {
	if len(values) == 0 {
		return 0
	}

	return float64(sumInts(values)) / float64(len(values))
}

func varianceInts(values []int) float64 {
	if len(values) == 0 {
		return 0
	}

	mean := averageInts(values)
	var sum float64
	for _, value := range values {
		diff := float64(value) - mean
		sum += diff * diff
	}

	return sum / float64(len(values))
}

func sumInts(values []int) int {
	total := 0
	for _, value := range values {
		total += value
	}

	return total
}
