package domain

import "fmt"

type Scenario struct {
	ID                    string
	InitialInventory      int
	InitialBacklog        int
	InitialPipelineGoods  []int
	InitialPipelineOrders []int
	ConsumerDemand        []int
	ShippingDelay         int
	OrderDelay            int
	ProductionDelay       int
	HoldingCost           int
	BacklogCost           int
}

func (s Scenario) Validate() error {
	if len(s.ConsumerDemand) == 0 {
		return ErrScenarioDemandEmpty
	}
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
	if err := validateNonNegativeSlice(s.ConsumerDemand); err != nil {
		return err
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

func validateNonNegativeSlice(values []int) error {
	for _, value := range values {
		if value < 0 {
			return ErrNegativeValue
		}
	}

	return nil
}
