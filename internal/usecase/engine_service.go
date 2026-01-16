package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/Victor-armando18/service-commercial/internal/domain"
	"github.com/Victor-armando18/service-commercial/internal/interfaces"
)

type EngineService struct {
	loader   interfaces.RulePackLoader
	executor interfaces.RuleExecutor
}

func NewEngineService(loader interfaces.RulePackLoader, executor interfaces.RuleExecutor) interfaces.EngineFacade {
	return &EngineService{loader: loader, executor: executor}
}

func (e *EngineService) RunEngine(ctx context.Context, initialOrder domain.Order, version string) (*domain.EngineResult, error) {
	rulePack, err := e.loader.Load(ctx, version)
	if err != nil {
		return nil, err
	}

	currentOrder := initialOrder
	e.prepareOrderData(&currentOrder)

	executionLog := []domain.ExecutionStep{}
	guardsHit := []domain.GuardViolation{}

	phases := []string{"baseline", "allocation", "taxes", "totals", "guards"}

	for _, phase := range phases {
		rules := e.getRules(rulePack.Rules, phase)
		for _, rule := range rules {
			out, err := e.executor.Execute(ctx, rule.Logic, map[string]interface{}{"order": currentOrder})
			if err != nil {
				continue
			}

			if phase == "guards" {
				if v, ok := out.(bool); ok && v {
					guardsHit = append(guardsHit, domain.GuardViolation{
						RuleID: rule.ID,
						Reason: fmt.Sprintf("Violation in %s phase", phase),
					})
				}
				continue
			}

			e.applyUpdate(rule.OutputKey, out, &currentOrder)
			executionLog = append(executionLog, domain.ExecutionStep{
				Phase:  phase,
				RuleID: rule.ID,
				Action: fmt.Sprintf("Update %s", rule.OutputKey),
			})
		}
	}

	return &domain.EngineResult{
		FinalOrder:   currentOrder,
		TotalValue:   currentOrder.BaseValue,
		GuardsHit:    guardsHit,
		ExecutionLog: executionLog,
	}, nil
}

func (e *EngineService) prepareOrderData(order *domain.Order) {
	var totalQty int
	var totalValue float64
	for _, item := range order.Items {
		totalQty += item.Qty
		totalValue += item.Value * float64(item.Qty)
	}
	order.TotalItems = totalQty
	if order.BaseValue == 0 {
		order.BaseValue = totalValue
	}
}

func (e *EngineService) getRules(rules []domain.RuleConfig, phase string) []domain.RuleConfig {
	var filtered []domain.RuleConfig
	for _, r := range rules {
		if r.Phase == phase {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func (e *EngineService) applyUpdate(key string, val interface{}, order *domain.Order) {
	fVal, _ := val.(float64)
	if key == "order.BaseValue" {
		order.BaseValue = fVal
	} else if strings.HasPrefix(key, "order.AppliedTaxes.") {
		tax := strings.TrimPrefix(key, "order.AppliedTaxes.")
		if order.AppliedTaxes == nil {
			order.AppliedTaxes = make(map[string]float64)
		}
		order.AppliedTaxes[tax] = fVal
	}
}
