package domain

import (
	"fmt"
	"time"
)

// --- Estruturas de Entrada/Saída ---

// Order representa o estado do pedido ou item a ser processado.
type Order struct {
	ID                 string
	BaseValue          float64
	Items              []OrderItem
	AppliedTaxes       map[string]float64 // Para rastrear impostos aplicados
	TotalItems         int                // Campo auxiliar para guardas
	DiscountPercentage float64
}

type OrderItem struct {
	SKU   string
	Value float64
	Qty   int
}

// EngineContext carrega o estado global necessário durante a execução do pipeline.
type EngineContext struct {
	Timestamp time.Time
	UserID    string
	Metadata  map[string]interface{}
}

// RulePackDefinition define a estrutura de um conjunto de regras carregado.
type RulePackDefinition struct {
	Version     string       `json:"version"`
	Rules       []RuleConfig `json:"rules"`
	Description string       `json:"description,omitempty"`
}

type RuleConfig struct {
	ID        string                 `json:"id"`
	Phase     string                 `json:"phase"`      // Ex: "baseline", "allocation", "tax", "guards"
	Logic     map[string]interface{} `json:"logic"`      // JsonLogic structure
	OutputKey string                 `json:"output_key"` // Onde armazenar o resultado (Ex: order.AppliedTaxes.VAT)
	ErrorMessage string      `json:"error_message,omitempty"`
}

type EngineResult struct {
	FinalOrder   Order
	TotalValue   float64
	GuardsHit    []GuardViolation
	ExecutionLog []ExecutionStep
}

type ExecutionStep struct {
	Phase  string
	RuleID string
	Action string
}

type GuardViolation struct {
	RuleID  string
	Reason  string
	Context string
}

// --- Constantes e Erros ---
var (
	// Erro definido no domínio, mas acessível via interfaces
	ErrRuleExecutionFailed = fmt.Errorf("rule execution failed")
)
