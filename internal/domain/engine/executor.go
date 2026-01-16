package engine

type RuleExecutor interface {
	Execute(rule Rule, ctx *EngineContext) error
}

type GuardExecutor interface {
	Execute(guard Guard, ctx *EngineContext) error
}
