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
	e.hydrateData(&currentOrder)

	executionLog := []domain.ExecutionStep{}
	guardsHit := []domain.GuardViolation{}

	phases := []string{"baseline", "allocation", "taxes", "totals", "guards"}

	for _, phase := range phases {
		rules := e.getRules(rulePack.Rules, phase)
		for _, rule := range rules {
			oldValue := e.getValue(rule.OutputKey, &currentOrder)

			out, err := e.executor.Execute(ctx, rule.Logic, map[string]interface{}{"order": currentOrder})
			if err != nil {
				continue
			}
			if phase == "guards" {
				if v, ok := out.(bool); ok && v {
					// Captura a mensagem personalizada da regra ou usa uma padrão
					msg := rule.ErrorMessage
					if msg == "" {
						msg = "Condição restritiva atingida"
					}

					guardsHit = append(guardsHit, domain.GuardViolation{
						RuleID:  rule.ID,
						Reason:  "Violation Detected",
						Context: msg, // Agora vem direto do JSON!
					})
				}
				continue
			}

			e.applyUpdate(rule.OutputKey, out, &currentOrder)

			newValue := e.getValue(rule.OutputKey, &currentOrder)

			executionLog = append(executionLog, domain.ExecutionStep{
				Phase:  phase,
				RuleID: rule.ID,
				Action: fmt.Sprintf("Changed %s: [%.2f -> %.2f]", rule.OutputKey, oldValue, newValue),
			})
		}
	}

	finalTotal := currentOrder.BaseValue
	for _, taxValue := range currentOrder.AppliedTaxes {
		finalTotal += taxValue
	}

	return &domain.EngineResult{
		FinalOrder:   currentOrder,
		TotalValue:   finalTotal,
		GuardsHit:    guardsHit,
		ExecutionLog: executionLog,
	}, nil
}

func (e *EngineService) hydrateData(order *domain.Order) {
	var q int
	var v float64
	for _, i := range order.Items {
		q += i.Qty
		v += i.Value * float64(i.Qty)
	}
	order.TotalItems = q
	if order.BaseValue == 0 {
		order.BaseValue = v
	}
}

func (e *EngineService) getRules(rules []domain.RuleConfig, phase string) []domain.RuleConfig {
	var f []domain.RuleConfig
	for _, r := range rules {
		if r.Phase == phase {
			f = append(f, r)
		}
	}
	return f
}

func (e *EngineService) getValue(key string, order *domain.Order) float64 {
	if key == "order.BaseValue" {
		return order.BaseValue
	}
	if strings.HasPrefix(key, "order.AppliedTaxes.") {
		k := strings.TrimPrefix(key, "order.AppliedTaxes.")
		if order.AppliedTaxes == nil {
			return 0
		}
		return order.AppliedTaxes[k]
	}
	return 0
}

func (e *EngineService) applyUpdate(key string, val interface{}, order *domain.Order) {
	var f float64
	switch v := val.(type) {
	case float64:
		f = v
	case int:
		f = float64(v)
	case int64:
		f = float64(v)
	default:
		return
	}

	if key == "order.BaseValue" {
		order.BaseValue = f
	} else if strings.HasPrefix(key, "order.AppliedTaxes.") {
		t := strings.TrimPrefix(key, "order.AppliedTaxes.")
		if order.AppliedTaxes == nil {
			order.AppliedTaxes = make(map[string]float64)
		}
		order.AppliedTaxes[t] = f
	}
}
