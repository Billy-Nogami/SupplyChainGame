package memory

import (
	"context"

	"supply-chain-simulator/internal/domain"
)

type ScenarioRepository struct {
	defaultScenario domain.Scenario
	scenarios       map[string]domain.Scenario
}

func NewScenarioRepository() *ScenarioRepository {
	defaultScenario := domain.Scenario{
		ID:                    "default-beer-game",
		InitialInventory:      12,
		InitialBacklog:        0,
		InitialPipelineGoods:  []int{4, 4},
		InitialPipelineOrders: []int{4},
		ShippingDelay:         2,
		OrderDelay:            1,
		ProductionDelay:       2,
		HoldingCost:           1,
		BacklogCost:           2,
		DemandMode:            domain.DemandModeRandomBlocks,
		DemandMin:             8,
		DemandMax:             20,
		DemandChangePeriod:    5,
	}

	return &ScenarioRepository{
		defaultScenario: defaultScenario,
		scenarios: map[string]domain.Scenario{
			defaultScenario.ID: defaultScenario,
		},
	}
}

func (r *ScenarioRepository) GetDefault(_ context.Context) (domain.Scenario, error) {
	return r.defaultScenario, nil
}

func (r *ScenarioRepository) GetByID(_ context.Context, scenarioID string) (domain.Scenario, error) {
	scenario, ok := r.scenarios[scenarioID]
	if !ok {
		return domain.Scenario{}, domain.ErrRoomNotFound
	}

	return scenario, nil
}
