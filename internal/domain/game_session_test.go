package domain

import (
	"testing"
	"time"
)

func TestNewGameSessionInitializesDefaultState(t *testing.T) {
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	scenario := testScenario()

	session, err := NewGameSession("session-1", "room-1", scenario, 30, now)
	if err != nil {
		t.Fatalf("NewGameSession() error = %v", err)
	}

	if session.Status != GameStatusActive {
		t.Fatalf("status = %s, want %s", session.Status, GameStatusActive)
	}
	if len(session.Nodes) != len(AllRoles) {
		t.Fatalf("nodes length = %d, want %d", len(session.Nodes), len(AllRoles))
	}
	if got := session.GoodsPipelines[RoleRetailer]; len(got) != scenario.ShippingDelay {
		t.Fatalf("retailer goods pipeline length = %d, want %d", len(got), scenario.ShippingDelay)
	}
	if got := session.OrderPipelines[RoleFactory]; len(got) != scenario.OrderDelay {
		t.Fatalf("factory order pipeline length = %d, want %d", len(got), scenario.OrderDelay)
	}
	if len(session.ProductionQueue) != scenario.ProductionDelay {
		t.Fatalf("production queue length = %d, want %d", len(session.ProductionQueue), scenario.ProductionDelay)
	}
}

func TestAdvanceWeekCalculatesBeerGameStep(t *testing.T) {
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	scenario := testScenario()
	session, err := NewGameSession("session-1", "room-1", scenario, 2, now)
	if err != nil {
		t.Fatalf("NewGameSession() error = %v", err)
	}

	weekState, err := session.AdvanceWeek(map[Role]int{
		RoleFactory:     4,
		RoleDistributor: 4,
		RoleWholesaler:  4,
		RoleRetailer:    4,
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("AdvanceWeek() error = %v", err)
	}

	if weekState.Week != 1 {
		t.Fatalf("week = %d, want 1", weekState.Week)
	}
	if weekState.TotalCost != 47 {
		t.Fatalf("total cost = %d, want 47", weekState.TotalCost)
	}

	retailer := nodeByRole(weekState.Nodes, RoleRetailer)
	if retailer.IncomingOrder != 5 {
		t.Fatalf("retailer incoming order = %d, want 5", retailer.IncomingOrder)
	}
	if retailer.IncomingGoods != 4 {
		t.Fatalf("retailer incoming goods = %d, want 4", retailer.IncomingGoods)
	}
	if retailer.Inventory != 11 {
		t.Fatalf("retailer inventory = %d, want 11", retailer.Inventory)
	}
	if retailer.ActualShipment != 5 {
		t.Fatalf("retailer shipment = %d, want 5", retailer.ActualShipment)
	}

	factory := nodeByRole(weekState.Nodes, RoleFactory)
	if factory.IncomingOrder != 4 {
		t.Fatalf("factory incoming order = %d, want 4", factory.IncomingOrder)
	}
	if factory.IncomingGoods != 4 {
		t.Fatalf("factory incoming goods = %d, want 4", factory.IncomingGoods)
	}
	if factory.Inventory != 12 {
		t.Fatalf("factory inventory = %d, want 12", factory.Inventory)
	}

	if got := session.GoodsPipelines[RoleDistributor]; !equalInts(got, []int{4, 4}) {
		t.Fatalf("distributor goods pipeline = %v, want [4 4]", got)
	}
	if got := session.OrderPipelines[RoleFactory]; !equalInts(got, []int{4}) {
		t.Fatalf("factory order pipeline = %v, want [4]", got)
	}
	if got := session.ProductionQueue; !equalInts(got, []int{4, 4}) {
		t.Fatalf("production queue = %v, want [4 4]", got)
	}
}

func TestAdvanceWeekFinishesSessionAtMaxWeeks(t *testing.T) {
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	session, err := NewGameSession("session-1", "room-1", testScenario(), 1, now)
	if err != nil {
		t.Fatalf("NewGameSession() error = %v", err)
	}

	_, err = session.AdvanceWeek(map[Role]int{
		RoleFactory:     4,
		RoleDistributor: 4,
		RoleWholesaler:  4,
		RoleRetailer:    4,
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("AdvanceWeek() error = %v", err)
	}

	if session.Status != GameStatusFinished {
		t.Fatalf("status = %s, want %s", session.Status, GameStatusFinished)
	}
	if _, err = session.AdvanceWeek(map[Role]int{
		RoleFactory:     4,
		RoleDistributor: 4,
		RoleWholesaler:  4,
		RoleRetailer:    4,
	}, now.Add(2*time.Minute)); err == nil {
		t.Fatal("AdvanceWeek() second call error = nil, want error")
	}
}

func TestScenarioMaterializeDemandGeneratesRandomBlocks(t *testing.T) {
	scenario := Scenario{
		ID:                    "scenario-random",
		InitialInventory:      12,
		InitialBacklog:        0,
		InitialPipelineGoods:  []int{4, 4},
		InitialPipelineOrders: []int{4},
		ShippingDelay:         2,
		OrderDelay:            1,
		ProductionDelay:       2,
		HoldingCost:           1,
		BacklogCost:           2,
		DemandMode:            DemandModeRandomBlocks,
		DemandMin:             8,
		DemandMax:             20,
		DemandChangePeriod:    5,
	}

	materialized, err := scenario.MaterializeDemand(15, 42)
	if err != nil {
		t.Fatalf("MaterializeDemand() error = %v", err)
	}

	if len(materialized.ConsumerDemand) != 15 {
		t.Fatalf("consumer demand length = %d, want 15", len(materialized.ConsumerDemand))
	}

	for blockStart := 0; blockStart < 15; blockStart += 5 {
		blockValue := materialized.ConsumerDemand[blockStart]
		if blockValue < 8 || blockValue > 20 {
			t.Fatalf("block value = %d, want in [8, 20]", blockValue)
		}

		blockEnd := blockStart + 5
		if blockEnd > 15 {
			blockEnd = 15
		}
		for i := blockStart; i < blockEnd; i++ {
			if materialized.ConsumerDemand[i] != blockValue {
				t.Fatalf("consumer demand[%d] = %d, want %d", i, materialized.ConsumerDemand[i], blockValue)
			}
		}
	}
}

func testScenario() Scenario {
	return Scenario{
		ID:                    "scenario-1",
		InitialInventory:      12,
		InitialBacklog:        0,
		InitialPipelineGoods:  []int{4, 4},
		InitialPipelineOrders: []int{4},
		ConsumerDemand:        []int{5, 5, 5, 5, 9, 9},
		ShippingDelay:         2,
		OrderDelay:            1,
		ProductionDelay:       2,
		HoldingCost:           1,
		BacklogCost:           2,
	}
}

func equalInts(left, right []int) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}

	return true
}
