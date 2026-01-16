package engine

type Engine struct {
	Rules  []Rule
	Guards []Guard
}

func (e *Engine) Run(ctx *EngineContext, re RuleExecutor, ge GuardExecutor) error {
	for _, r := range e.Rules {
		ctx.Phase = r.Phase
		if err := re.Execute(r, ctx); err != nil {
			return err
		}
	}

	for _, g := range e.Guards {
		if err := ge.Execute(g, ctx); err != nil {
			return err
		}
	}

	return nil
}
