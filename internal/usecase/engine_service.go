package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Victor-armando18/service-commercial/internal/domain"
	"github.com/Victor-armando18/service-commercial/internal/interfaces"
	jsonpatch "github.com/evanphx/json-patch/v5"
)

type EngineService struct {
	loader   interfaces.RulePackLoader
	executor interfaces.RuleExecutor
}

func NewEngineService(loader interfaces.RulePackLoader, executor interfaces.RuleExecutor) interfaces.EngineFacade {
	return &EngineService{loader: loader, executor: executor}
}

func (e *EngineService) RunEngine(ctx context.Context, initialOrder domain.Order, version string) (*domain.EngineResult, error) {
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	rulePack, err := e.loader.Load(ctx, version)
	if err != nil {
		return nil, err
	}

	// Preservação total da cópia original
	workingOrder := initialOrder
	workingOrder.RulesVersion = version
	e.hydrateData(&workingOrder)

	initialJSON, _ := json.Marshal(initialOrder)
	executionLog := []domain.ExecutionStep{}
	guardsHit := []domain.GuardViolation{}

	phases := []string{"baseline", "orderAdjust", "allocation", "taxes", "totals", "guards"}

	for _, phase := range phases {
		rules := e.getRules(rulePack.Rules, phase)
		for _, rule := range rules {
			out, err := e.executor.Execute(ctx, rule.Logic, map[string]interface{}{"order": workingOrder})
			if err != nil || out == nil {
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

	// Cálculo Final
	var taxTotal float64
	for _, val := range workingOrder.AppliedTaxes {
		taxTotal += val
	}
	workingOrder.TotalValue = workingOrder.BaseValue + taxTotal

	// Geração do Fragmento (Mantém Currency e CorrelationID se estiverem na workingOrder)
	finalJSON, _ := json.Marshal(workingOrder)
	patch, _ := jsonpatch.CreateMergePatch(initialJSON, finalJSON)

	var stateFragment map[string]interface{}
	json.Unmarshal(finalJSON, &stateFragment)

	return &domain.EngineResult{
		StateFragment: stateFragment,
		ServerDelta:   len(patch) > 2,
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
	// Só calculamos o BaseValue se o motor ainda não o definiu via regras
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

func (e *EngineService) applyUpdate(key string, val interface{}, order *domain.Order) {
	f, ok := e.toFloat(val)
	if !ok {
		return
	}

	switch {
	case key == "order.baseValue":
		order.BaseValue = f
	case strings.HasPrefix(key, "order.appliedTaxes."):
		taxName := strings.TrimPrefix(key, "order.appliedTaxes.")
		if order.AppliedTaxes == nil {
			order.AppliedTaxes = make(map[string]float64)
		}
		order.AppliedTaxes[taxName] = f
	case key == "order.totalValue": // Adicionado para suporte a regras de totalização customizadas
		order.TotalValue = f
	}
}

func (e *EngineService) toFloat(i interface{}) (float64, bool) {
	switch v := i.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case json.Number:
		f, _ := v.Float64()
		return f, true
	}
	return 0, false
}
