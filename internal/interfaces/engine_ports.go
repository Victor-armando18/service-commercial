package interfaces

import (
	"context"

	"github.com/Victor-armando18/service-commercial/internal/domain"
)

// Definimos o erro aqui para que o usecase possa referenciá-lo facilmente
var ErrRuleExecutionFailed = domain.ErrRuleExecutionFailed

// RulePackLoader define o contrato para carregar os RulePacks (de disco, rede, etc.).
type RulePackLoader interface {
	Load(ctx context.Context, version string) (*domain.RulePackDefinition, error)
}

// RuleExecutor define o contrato para executar uma regra JsonLogic com operadores customizados.
type RuleExecutor interface {
	Execute(ctx context.Context, ruleData map[string]interface{}, contextVars map[string]interface{}) (interface{}, error)
	RegisterCustomOperator(name string, logic func(args ...interface{}) interface{})
}

// EngineFacade é a função clara exposta ao mundo externo (a porta de entrada da aplicação).
type EngineFacade interface {
	RunEngine(ctx context.Context, initialOrder domain.Order, rulePackVersion string) (*domain.EngineResult, error)
}
