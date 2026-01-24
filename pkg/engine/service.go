package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	jsonpatch "github.com/evanphx/json-patch/v5"
)

type EngineService struct {
	loader   RulePackLoader
	executor *JsonLogicExecutor
}

func NewEngineService(l RulePackLoader, e *JsonLogicExecutor) *EngineService {
	return &EngineService{loader: l, executor: e}
}

func (e *EngineService) RunEngine(ctx context.Context, initialOrder Order, version string) (*EngineResult, error) {
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	rulePack, err := e.loader.Load(ctx, version)
	if err != nil {
		return nil, err
	}

	workingOrder := initialOrder
	workingOrder.RulesVersion = version
	e.hydrateData(&workingOrder)

	initialJSON, _ := json.Marshal(initialOrder)
	executionLog := []ExecutionStep{}
	guardsHit := []GuardViolation{}

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
					guardsHit = append(guardsHit, GuardViolation{
						RuleID:  rule.ID,
						Reason:  "Violation Detected",
						Context: rule.ErrorMessage,
					})
				}
				continue
			}

			e.applyUpdate(rule.OutputKey, out, &workingOrder)
			executionLog = append(executionLog, ExecutionStep{
				Phase:   phase,
				RuleID:  rule.ID,
				Action:  "compute",
				Message: fmt.Sprintf("Updated %s", rule.OutputKey),
			})
		}
	}

	var taxTotal float64
	for _, val := range workingOrder.AppliedTaxes {
		taxTotal += val
	}
	workingOrder.TotalValue = workingOrder.BaseValue + taxTotal

	finalJSON, _ := json.Marshal(workingOrder)
	patch, _ := jsonpatch.CreateMergePatch(initialJSON, finalJSON)

	var stateFragment map[string]interface{}
	json.Unmarshal(finalJSON, &stateFragment)

	return &EngineResult{
		StateFragment: stateFragment,
		ServerDelta:   len(patch) > 2,
		RulesVersion:  version,
		ExecutionLog:  executionLog,
		GuardsHit:     guardsHit,
	}, nil
}

func (e *EngineService) hydrateData(order *Order) {
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

func (e *EngineService) getRules(rules []RuleConfig, phase string) []RuleConfig {
	var f []RuleConfig
	for _, r := range rules {
		if r.Phase == phase {
			f = append(f, r)
		}
	}
	return f
}

func (e *EngineService) applyUpdate(key string, val interface{}, order *Order) {
	f, ok := anyToFloat(val)
	if !ok {
		return
	}

	switch {
	case key == "order.baseValue":
		order.BaseValue = f
	case strings.HasPrefix(key, "order.appliedTaxes."):
		t := strings.TrimPrefix(key, "order.appliedTaxes.")
		if order.AppliedTaxes == nil {
			order.AppliedTaxes = make(map[string]float64)
		}
		order.AppliedTaxes[t] = f
	case key == "order.totalValue":
		order.TotalValue = f
	}
}
