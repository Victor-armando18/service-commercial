package runengine

import (
	"context"

	"service-commercial/internal/domain/engine"
	"service-commercial/internal/domain/model"
)

type UseCase struct {
	RuleExecutor  engine.RuleExecutor
	GuardExecutor engine.GuardExecutor
	Differ        Differ
}

type Differ interface {
	Diff(before, after map[string]any) map[string]any
}

func (u *UseCase) Run(ctx context.Context, order model.Order, pack engine.RulePack) (engine.Result, error) {
	state := map[string]any{
		"order":  order,
		"totals": map[string]any{},
	}

	engineCtx := &engine.EngineContext{State: state}

	e := &engine.Engine{Rules: pack.Rules, Guards: pack.Guards}
	before := clone(state)

	if err := e.Run(engineCtx, u.RuleExecutor, u.GuardExecutor); err != nil {
		return engine.Result{}, err
	}

	delta := u.Differ.Diff(before, engineCtx.State)

	return engine.Result{
		StateFragment: engineCtx.State,
		Delta:         delta,
		Reasons:       engineCtx.Reasons,
		RulesVersion:  pack.Version,
	}, nil
}

func clone(src map[string]any) map[string]any {
	dst := map[string]any{}
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
