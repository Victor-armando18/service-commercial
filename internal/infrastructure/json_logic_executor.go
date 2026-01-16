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
	// 1. Verificar se é um operador customizado direto
	for opName, fn := range j.customOps {
		if args, ok := ruleData[opName]; ok {
			return j.handleManualEval(args, contextVars, fn), nil
		}
	}

	// 2. Preparação do Contexto (Crucial para o JsonLogic)
	// Convertemos a struct Order para um mapa puro com chaves JSON minúsculas
	var cleanContext map[string]interface{}
	rawContext, _ := json.Marshal(contextVars)
	json.Unmarshal(rawContext, &cleanContext)

	ruleJSON, _ := json.Marshal(ruleData)

	var resultBuffer bytes.Buffer
	// Aplicamos o JsonLogic usando o contexto limpo
	err := jsonlogic.Apply(bytes.NewReader(ruleJSON), bytes.NewReader(rawContext), &resultBuffer)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", interfaces.ErrRuleExecutionFailed, err)
	}

	resultStr := resultBuffer.String()
	if resultStr == "" || resultStr == "null" {
		return nil, nil
	}

	// 3. Decodificação do resultado com UseNumber para manter precisão decimal
	var res interface{}
	decoder := json.NewDecoder(strings.NewReader(resultStr))
	decoder.UseNumber()
	if err := decoder.Decode(&res); err != nil {
		return nil, err
	}

	// Converter json.Number para float64 ou bool conforme necessário
	return j.finalizeValue(res), nil
}

func (j *JsonLogicExecutor) finalizeValue(val interface{}) interface{} {
	if n, ok := val.(json.Number); ok {
		if f, err := n.Float64(); err == nil {
			return f
		}
	}
	return val
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

	// Normalização profunda para busca manual
	var cleanData map[string]interface{}
	b, _ := json.Marshal(data)
	json.Unmarshal(b, &cleanData)

	order, _ := cleanData["order"].(map[string]interface{})
	if order == nil {
		return 0.0
	}

	// Navegação por pontos (ex: order.baseValue ou order.appliedTaxes.VAT)
	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return 0.0
	}

	var current interface{} = order
	for i := 1; i < len(parts); i++ {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[parts[i]]
		} else {
			return 0.0
		}
	}

	if n, ok := current.(json.Number); ok {
		f, _ := n.Float64()
		return f
	}
	if f, ok := current.(float64); ok {
		return f
	}

	return current
}

func CustomRound(args ...interface{}) interface{} {
	if len(args) == 0 {
		return 0.0
	}
	v := reflectToFloat(args[0])
	return math.Round(v)
}

func CustomAllocate(args ...interface{}) interface{} {
	if len(args) < 2 {
		return 0.0
	}
	val := reflectToFloat(args[0])
	parts := reflectToFloat(args[1])
	if parts == 0 {
		return 0.0
	}
	return val / parts
}

func reflectToFloat(i interface{}) float64 {
	switch v := i.(type) {
	case float64:
		return v
	case json.Number:
		f, _ := v.Float64()
		return f
	case int:
		return float64(v)
	}
	return 0.0
}
