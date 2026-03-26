package domain

import (
	"fmt"
	"math/rand"
)

type DemandMode string

const (
	DemandModeFixed        DemandMode = "fixed"
	DemandModeRandomBlocks DemandMode = "random_blocks"
)

type Scenario struct {
	ID                    string     `json:"id"`
	InitialInventory      int        `json:"initial_inventory"`
	InitialBacklog        int        `json:"initial_backlog"`
	InitialPipelineGoods  []int      `json:"initial_pipeline_goods"`
	InitialPipelineOrders []int      `json:"initial_pipeline_orders"`
	ConsumerDemand        []int      `json:"consumer_demand"`
	ShippingDelay         int        `json:"shipping_delay"`
	OrderDelay            int        `json:"order_delay"`
	ProductionDelay       int        `json:"production_delay"`
	HoldingCost           int        `json:"holding_cost"`
	BacklogCost           int        `json:"backlog_cost"`
	DemandMode            DemandMode `json:"demand_mode"`
	DemandMin             int        `json:"demand_min"`
	DemandMax             int        `json:"demand_max"`
	DemandChangePeriod    int        `json:"demand_change_period"`
}

func (s Scenario) Validate() error {
	if s.InitialInventory < 0 || s.InitialBacklog < 0 || s.HoldingCost < 0 || s.BacklogCost < 0 {
		return ErrNegativeValue
	}
	if s.ShippingDelay <= 0 || s.OrderDelay <= 0 || s.ProductionDelay <= 0 {
		return ErrInvalidDelay
	}
	if err := validateNonNegativeSlice(s.InitialPipelineGoods); err != nil {
		return err
	}
	if err := validateNonNegativeSlice(s.InitialPipelineOrders); err != nil {
		return err
	}
	switch s.normalizedDemandMode() {
	case DemandModeFixed:
		if len(s.ConsumerDemand) == 0 {
			return ErrScenarioDemandEmpty
		}
		if err := validateNonNegativeSlice(s.ConsumerDemand); err != nil {
			return err
		}
	case DemandModeRandomBlocks:
		if s.DemandMin < 0 || s.DemandMax < 0 {
			return ErrNegativeValue
		}
		if s.DemandMax < s.DemandMin {
			return fmt.Errorf("demand_max must be greater than or equal to demand_min")
		}
		if s.DemandChangePeriod <= 0 {
			return fmt.Errorf("demand_change_period must be positive")
		}
	default:
		return fmt.Errorf("unsupported demand mode: %s", s.DemandMode)
	}
	if len(s.InitialPipelineGoods) != s.ShippingDelay {
		return fmt.Errorf("initial goods pipeline length must match shipping delay")
	}
	if len(s.InitialPipelineOrders) != s.OrderDelay {
		return fmt.Errorf("initial order pipeline length must match order delay")
	}
	if len(s.InitialPipelineGoods) != s.ProductionDelay {
		return fmt.Errorf("initial production pipeline length must match production delay")
	}

	return nil
}

func (s Scenario) DemandForWeek(week int) int {
	if week <= 0 {
		return 0
	}
	if week <= len(s.ConsumerDemand) {
		return s.ConsumerDemand[week-1]
	}

	return s.ConsumerDemand[len(s.ConsumerDemand)-1]
}

func (s Scenario) MaterializeDemand(maxWeeks int, seed int64) (Scenario, error) {
	if err := s.Validate(); err != nil {
		return Scenario{}, err
	}
	if maxWeeks <= 0 {
		return Scenario{}, ErrInvalidMaxWeeks
	}

	materialized := s
	switch s.normalizedDemandMode() {
	case DemandModeFixed:
		materialized.ConsumerDemand = extendDemand(copyInts(s.ConsumerDemand), maxWeeks)
	case DemandModeRandomBlocks:
		materialized.ConsumerDemand = generateRandomBlockDemand(maxWeeks, s.DemandMin, s.DemandMax, s.DemandChangePeriod, seed)
	}

	return materialized, nil
}

func (s Scenario) normalizedDemandMode() DemandMode {
	if s.DemandMode == "" {
		return DemandModeFixed
	}

	return s.DemandMode
}

func validateNonNegativeSlice(values []int) error {
	for _, value := range values {
		if value < 0 {
			return ErrNegativeValue
		}
	}

	return nil
}

func extendDemand(values []int, maxWeeks int) []int {
	if len(values) >= maxWeeks {
		return values[:maxWeeks]
	}

	result := make([]int, 0, maxWeeks)
	result = append(result, values...)
	last := values[len(values)-1]
	for len(result) < maxWeeks {
		result = append(result, last)
	}

	return result
}

func generateRandomBlockDemand(maxWeeks, minDemand, maxDemand, changePeriod int, seed int64) []int {
	rng := rand.New(rand.NewSource(seed))
	demand := make([]int, maxWeeks)

	for start := 0; start < maxWeeks; start += changePeriod {
		value := minDemand
		if maxDemand > minDemand {
			value += rng.Intn(maxDemand - minDemand + 1)
		}

		end := start + changePeriod
		if end > maxWeeks {
			end = maxWeeks
		}

		for i := start; i < end; i++ {
			demand[i] = value
		}
	}

	return demand
}
