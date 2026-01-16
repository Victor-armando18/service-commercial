package jsonlogic

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"service-commercial/internal/domain/engine"
)

type Executor struct{}
type GuardExecutor struct{}

func (e *Executor) Execute(rule engine.Rule, ctx *engine.EngineContext) error {
	_, err := eval(rule.Logic, ctx)
	if err == nil {
		ctx.Reasons = append(ctx.Reasons, engine.Reason{
			RuleID: rule.ID,
			Phase:  ctx.Phase,
			Why:    "rule applied",
		})
	}
	return err
}

func (g *GuardExecutor) Execute(guard engine.Guard, ctx *engine.EngineContext) error {
	v, err := eval(guard.Logic, ctx)
	if err != nil {
		return err
	}
	b, ok := v.(bool)
	if !ok {
		return errors.New("guard must return boolean")
	}
	if !b {
		return errors.New(guard.Reason)
	}
	return nil
}

func eval(expr any, ctx *engine.EngineContext) (any, error) {
	switch v := expr.(type) {

	case map[string]any:
		for op, raw := range v {

			switch op {

			case "set":
				args := raw.([]any)
				path := args[0].(string)
				val, err := eval(args[1], ctx)
				if err != nil {
					return nil, err
				}
				setPath(ctx.State, path, val)
				return val, nil

			case "+", "-", "*", "/":
				return binary(op, raw, ctx)

			case ">":
				a, b, err := two(raw, ctx)
				if err != nil {
					return nil, err
				}
				return a > b, nil

			case "sum":
				src, err := eval(raw.([]any)[0], ctx)
				if err != nil || src == nil {
					return 0.0, nil
				}

				total := 0.0
				rv := reflect.ValueOf(src)
				if rv.Kind() == reflect.Slice {
					for i := 0; i < rv.Len(); i++ {
						total += toFloat(rv.Index(i).Interface())
					}
				}
				return total, nil

			case "foreach":
				cfg := raw.(map[string]any)
				src, err := eval(cfg["var"], ctx)
				if err != nil || src == nil {
					return []any{}, nil
				}

				rv := reflect.ValueOf(src)
				if rv.Kind() != reflect.Slice {
					return []any{}, nil
				}

				out := make([]any, 0, rv.Len())

				for i := 0; i < rv.Len(); i++ {
					local := clone(ctx.State)
					local["$"] = rv.Index(i).Interface()

					val, err := eval(cfg["do"], &engine.EngineContext{State: local})
					if err != nil {
						return nil, err
					}
					out = append(out, val)
				}
				return out, nil
			}
		}

	case string:
		if strings.HasPrefix(v, "$.") {
			return getPath(ctx.State, v), nil
		}
		return v, nil

	case float64, bool, int:
		return v, nil
	}

	return nil, fmt.Errorf("invalid expression")
}

func binary(op string, raw any, ctx *engine.EngineContext) (any, error) {
	a, b, err := two(raw, ctx)
	if err != nil {
		return nil, err
	}
	switch op {
	case "+":
		return a + b, nil
	case "-":
		return a - b, nil
	case "*":
		return a * b, nil
	case "/":
		return a / b, nil
	}
	return nil, nil
}

func two(raw any, ctx *engine.EngineContext) (float64, float64, error) {
	args := raw.([]any)
	av, err := eval(args[0], ctx)
	if err != nil {
		return 0, 0, err
	}
	bv, err := eval(args[1], ctx)
	if err != nil {
		return 0, 0, err
	}
	return toFloat(av), toFloat(bv), nil
}

func toFloat(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	default:
		return 0
	}
}

func getPath(root any, path string) any {
	parts := strings.Split(strings.TrimPrefix(path, "$."), ".")

	cur := root

	// Se existir $ no contexto e for map ou struct, usa como base
	if m, ok := root.(map[string]any); ok {
		if scoped, exists := m["$"]; exists {
			cur = scoped
		}
	} else if reflect.ValueOf(root).Kind() == reflect.Struct {
		v := reflect.ValueOf(root)
		if f := v.FieldByName("$"); f.IsValid() {
			cur = f.Interface()
		}
	}

	rv := reflect.ValueOf(cur)
	for _, p := range parts {
		if rv.Kind() == reflect.Pointer {
			rv = rv.Elem()
		}

		switch rv.Kind() {
		case reflect.Map:
			rv = rv.MapIndex(reflect.ValueOf(p))
		case reflect.Struct:
			rv = rv.FieldByName(p)
		default:
			return nil
		}

		if !rv.IsValid() {
			return nil
		}
	}

	return rv.Interface()
}

func setPath(root map[string]any, path string, val any) {
	parts := strings.Split(strings.TrimPrefix(path, "$."), ".")
	cur := root
	for i := 0; i < len(parts)-1; i++ {
		if _, ok := cur[parts[i]]; !ok {
			cur[parts[i]] = map[string]any{}
		}
		cur = cur[parts[i]].(map[string]any)
	}
	cur[parts[len(parts)-1]] = val
}

func clone(src map[string]any) map[string]any {
	dst := map[string]any{}
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
