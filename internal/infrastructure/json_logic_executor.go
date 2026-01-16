package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/Victor-armando18/service-commercial/internal/interfaces"
	"github.com/diegoholiveira/jsonlogic/v3"
)

type JsonLogicExecutor struct {
	customOps map[string]func(args ...interface{}) interface{}
}

func NewJsonLogicExecutor() *JsonLogicExecutor {
	return &JsonLogicExecutor{
		customOps: make(map[string]func(args ...interface{}) interface{}),
	}
}

func (j *JsonLogicExecutor) RegisterCustomOperator(name string, logic func(args ...interface{}) interface{}) {
	j.customOps[name] = logic
}

func (j *JsonLogicExecutor) Execute(ctx context.Context, ruleData map[string]interface{}, contextVars map[string]interface{}) (interface{}, error) {
	// 1. Tentar execução por Operadores Customizados (Customização manual)
	for opName, fn := range j.customOps {
		if args, ok := ruleData[opName]; ok {
			return j.handleManualEval(args, contextVars, fn), nil
		}
	}

	// 2. Execução Standard JsonLogic
	ruleJSON, _ := json.Marshal(ruleData)
	dataJSON, _ := json.Marshal(contextVars)
	var resultBuffer bytes.Buffer

	err := jsonlogic.Apply(strings.NewReader(string(ruleJSON)), strings.NewReader(string(dataJSON)), &resultBuffer)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", interfaces.ErrRuleExecutionFailed, err)
	}

	var res interface{}
	if resultBuffer.Len() == 0 || resultBuffer.String() == "null" {
		return nil, nil
	}
	json.Unmarshal(resultBuffer.Bytes(), &res)
	return res, nil
}

func (j *JsonLogicExecutor) handleManualEval(args interface{}, data map[string]interface{}, fn func(args ...interface{}) interface{}) interface{} {
	var params []interface{}
	if v, ok := args.([]interface{}); ok {
		for _, arg := range v {
			params = append(params, j.resolveVar(arg, data))
		}
	} else {
		params = append(params, j.resolveVar(args, data))
	}
	return fn(params...)
}

func (j *JsonLogicExecutor) resolveVar(arg interface{}, data map[string]interface{}) interface{} {
	m, ok := arg.(map[string]interface{})
	if !ok {
		return arg
	}
	path, ok := m["var"].(string)
	if !ok {
		return arg
	}

	order, _ := data["order"].(map[string]interface{})

	switch path {
	case "order.BaseValue":
		return order["BaseValue"]
	case "order.DiscountPercentage":
		return order["DiscountPercentage"]
	case "order.TotalItems":
		return order["TotalItems"]
	case "order.ID":
		return order["ID"]
	}

	// Resolução de impostos aninhados
	if strings.HasPrefix(path, "order.AppliedTaxes.") {
		taxKey := strings.TrimPrefix(path, "order.AppliedTaxes.")
		if taxes, ok := order["AppliedTaxes"].(map[string]float64); ok {
			return taxes[taxKey]
		}
	}
	return 0.0
}

func CustomRound(args ...interface{}) interface{} {
	if len(args) == 0 {
		return 0.0
	}
	if v, ok := args[0].(float64); ok {
		return math.Round(v)
	}
	return args[0]
}

func CustomAllocate(args ...interface{}) interface{} {
	if len(args) < 2 {
		return 0.0
	}
	val, _ := args[0].(float64)
	parts, _ := args[1].(float64)
	if parts == 0 {
		return 0.0
	}
	return val / parts
}
