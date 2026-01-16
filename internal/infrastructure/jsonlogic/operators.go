package jsonlogic

import (
	"math"
	"reflect"
)

func Sum(args ...any) float64 {
	s := 0.0
	for _, a := range args {
		s += toFloat64(a)
	}
	return s
}

func Round(value any, precision any) float64 {
	p := int(toFloat64(precision))
	f := math.Pow(10, float64(p))
	return math.Round(toFloat64(value)*f) / f
}

func Allocate(total any, weights any) []float64 {
	wt := reflect.ValueOf(weights)
	if wt.Kind() != reflect.Slice {
		return nil
	}
	sum := 0.0
	for i := 0; i < wt.Len(); i++ {
		sum += toFloat64(wt.Index(i).Interface())
	}
	res := make([]float64, wt.Len())
	for i := 0; i < wt.Len(); i++ {
		res[i] = toFloat64(total) * toFloat64(wt.Index(i).Interface()) / sum
	}
	return res
}

func toFloat64(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case int32:
		return float64(val)
	case uint:
		return float64(val)
	case uint64:
		return float64(val)
	case uint32:
		return float64(val)
	case bool:
		if val {
			return 1
		} else {
			return 0
		}
	default:
		return 0
	}
}
