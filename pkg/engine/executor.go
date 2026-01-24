package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"strings"

	"github.com/diegoholiveira/jsonlogic/v3"
)

type JsonLogicExecutor struct {
	customOps map[string]func(args ...interface{}) interface{}
}

func NewJsonLogicExecutor() *JsonLogicExecutor {
	j := &JsonLogicExecutor{
		customOps: make(map[string]func(args ...interface{}) interface{}),
	}
	j.RegisterCustomOperator("round", CustomRound)
	j.RegisterCustomOperator("allocate", CustomAllocate)
	return j
}

func (j *JsonLogicExecutor) RegisterCustomOperator(name string, logic func(args ...interface{}) interface{}) {
	j.customOps[name] = logic
}

func (j *JsonLogicExecutor) Execute(ctx context.Context, ruleData map[string]interface{}, contextVars map[string]interface{}) (interface{}, error) {
	if _, ok := ruleData["foreach"]; ok {
		return j.handleForeach(ruleData["foreach"], contextVars), nil
	}

	for opName, fn := range j.customOps {
		if args, ok := ruleData[opName]; ok {
			return j.handleManualEval(args, contextVars, fn), nil
		}
	}

	ruleJSON, _ := json.Marshal(ruleData)
	dataJSON, _ := json.Marshal(contextVars)

	var resultBuffer bytes.Buffer
	err := jsonlogic.Apply(bytes.NewReader(ruleJSON), bytes.NewReader(dataJSON), &resultBuffer)
	if err != nil {
		return nil, err
	}

	resultStr := resultBuffer.String()
	if resultStr == "" || resultStr == "null" {
		return nil, nil
	}

	var res interface{}
	decoder := json.NewDecoder(strings.NewReader(resultStr))
	decoder.UseNumber()
	decoder.Decode(&res)

	return j.finalizeValue(res), nil
}

func (j *JsonLogicExecutor) handleForeach(args interface{}, data map[string]interface{}) interface{} {
	params, ok := args.([]interface{})
	if !ok || len(params) < 2 {
		return 0.0
	}

	collection := j.resolveVar(params[0], data)
	var items []interface{}
	b, _ := json.Marshal(collection)
	json.Unmarshal(b, &items)

	logic, ok := params[1].(map[string]interface{})
	if !ok {
		return 0.0
	}

	var total float64
	for _, item := range items {
		itemCtx := map[string]interface{}{
			"item":  item,
			"order": data["order"],
		}
		res, _ := j.Execute(context.Background(), logic, itemCtx)
		if f, ok := anyToFloat(res); ok {
			total += f
		}
	}
	return total
}

func (j *JsonLogicExecutor) handleManualEval(args interface{}, data map[string]interface{}, fn func(args ...interface{}) interface{}) interface{} {
	var params []interface{}
	if list, ok := args.([]interface{}); ok {
		for _, item := range list {
			if subRule, isRule := item.(map[string]interface{}); isRule {
				res, _ := j.Execute(context.Background(), subRule, data)
				params = append(params, res)
			} else {
				params = append(params, j.resolveVar(item, data))
			}
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
	parts := strings.Split(path, ".")
	var current interface{} = data
	for _, part := range parts {
		tempBytes, _ := json.Marshal(current)
		var tempMap map[string]interface{}
		json.Unmarshal(tempBytes, &tempMap)
		current = tempMap[part]
		if current == nil {
			break
		}
	}
	return j.finalizeValue(current)
}

func (j *JsonLogicExecutor) finalizeValue(val interface{}) interface{} {
	if n, ok := val.(json.Number); ok {
		if f, err := n.Float64(); err == nil {
			return f
		}
	}
	return val
}

func CustomRound(args ...interface{}) interface{} {
	if len(args) == 0 {
		return 0.0
	}
	val, _ := anyToFloat(args[0])
	precision := 0
	if len(args) > 1 {
		if p, ok := anyToFloat(args[1]); ok {
			precision = int(p)
		}
	}
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

func CustomAllocate(args ...interface{}) interface{} {
	if len(args) < 2 {
		return 0.0
	}
	val, _ := anyToFloat(args[0])
	parts, _ := anyToFloat(args[1])
	if parts == 0 {
		return 0.0
	}
	return val / parts
}

func anyToFloat(i interface{}) (float64, bool) {
	switch v := i.(type) {
	case float64:
		return v, true
	case json.Number:
		f, _ := v.Float64()
		return f, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	}
	return 0, false
}
