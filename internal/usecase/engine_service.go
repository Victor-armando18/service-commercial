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
	// Garantir que a versão tem o prefixo "v"
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	rulePack, err := e.loader.Load(ctx, version)
	if err != nil {
		return nil, err
	}

	workingOrder := initialOrder
	e.hydrateData(&workingOrder)

	// Snapshot inicial dos campos que rastreamos
	initialSnapshot := e.takeSnapshot(workingOrder)

	executionLog := []domain.ExecutionStep{}
	guardsHit := []domain.GuardViolation{}

	// IMPORTANTE: Estas fases devem existir no JSON
	phases := []string{"baseline", "orderAdjust", "allocation", "taxes", "totals", "guards"}

	for _, phase := range phases {
		rules := e.getRules(rulePack.Rules, phase)
		for _, rule := range rules {
			out, err := e.executor.Execute(ctx, rule.Logic, map[string]interface{}{"order": workingOrder})
			if err != nil {
				continue
			}

			if phase == "guards" {
				if v, ok := out.(bool); ok && v {
					guardsHit = append(guardsHit, domain.GuardViolation{
						RuleID:  rule.ID,
						Reason:  "Violation Detected",
						Context: rule.ErrorMessage,
					})
				}
				continue
			}

			e.applyUpdate(rule.OutputKey, out, &workingOrder)
			executionLog = append(executionLog, domain.ExecutionStep{
				Phase:   phase,
				RuleID:  rule.ID,
				Action:  "compute",
				Message: fmt.Sprintf("Updated %s", rule.OutputKey),
			})
		}
	}

	// Recálculo autoritativo do Total
	finalTotal := workingOrder.BaseValue
	for _, v := range workingOrder.AppliedTaxes {
		finalTotal += v
	}
	workingOrder.TotalValue = finalTotal

	finalSnapshot := e.takeSnapshot(workingOrder)
	fragment := e.generateStateFragment(initialSnapshot, finalSnapshot)

	return &domain.EngineResult{
		StateFragment: fragment,
		ServerDelta:   len(fragment) > 0,
		RulesVersion:  version,
		ExecutionLog:  executionLog,
		GuardsHit:     guardsHit,
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
	// O BaseValue inicial é a soma bruta dos itens
	order.BaseValue = v
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

func (e *EngineService) applyUpdate(key string, val interface{}, order *domain.Order) {
	var f float64
	switch v := val.(type) {
	case float64:
		f = v
	case int:
		f = float64(v)
	default:
		return
	}

	// Alinhado com as tags JSON e chaves do ficheiro de regras
	if key == "order.baseValue" {
		order.BaseValue = f
	} else if strings.HasPrefix(key, "order.appliedTaxes.") {
		t := strings.TrimPrefix(key, "order.appliedTaxes.")
		if order.AppliedTaxes == nil {
			order.AppliedTaxes = make(map[string]float64)
		}
		order.AppliedTaxes[t] = f
	}
}

func (e *EngineService) takeSnapshot(o domain.Order) map[string]interface{} {
	snap := make(map[string]interface{})
	snap["baseValue"] = o.BaseValue
	snap["totalValue"] = o.TotalValue
	for k, v := range o.AppliedTaxes {
		snap["appliedTaxes."+k] = v
	}
	return snap
}

func (e *EngineService) generateStateFragment(before, after map[string]interface{}) map[string]interface{} {
	fragment := make(map[string]interface{})
	for k, vAfter := range after {
		if vBefore, ok := before[k]; !ok || vBefore != vAfter {
			fragment[k] = vAfter
		}
	}
	return fragment
}
