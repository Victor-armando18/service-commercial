package interfaces

import (
	"context"

	"service-commercial/internal/domain/engine"
	"service-commercial/internal/domain/model"
	"service-commercial/internal/infrastructure/diff"
	"service-commercial/internal/infrastructure/jsonlogic"
	"service-commercial/internal/usecase/runengine"
)

func RunEngine(ctx context.Context, order model.Order, pack engine.RulePack) (engine.Result, error) {
	uc := &runengine.UseCase{
		RuleExecutor:  &jsonlogic.Executor{},
		GuardExecutor: &jsonlogic.GuardExecutor{},
		Differ:        &diff.Differ{},
	}
	return uc.Run(ctx, order, pack)
}
