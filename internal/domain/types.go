package domain

import (
	"fmt"
	"time"
)

type Order struct {
	ID                 string             `json:"id"`
	Currency           string             `json:"currency"` // AOA, USD, BRL, EUR
	BaseValue          float64            `json:"baseValue"`
	Items              []OrderItem        `json:"items"`
	AppliedTaxes       map[string]float64 `json:"appliedTaxes"`
	TotalItems         int                `json:"totalItems"`
	DiscountPercentage float64            `json:"discountPercentage"`
	TotalValue         float64            `json:"totalValue"`
	RulesVersion       string             `json:"rulesVersion"`
	CorrelationID      string             `json:"correlationId"`
}

type OrderItem struct {
	SKU   string  `json:"sku"`
	Value float64 `json:"value"`
	Qty   int     `json:"qty"`
}

type PatchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
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
	ID           string                 `json:"id"`
	Phase        string                 `json:"phase"`      // Ex: "baseline", "allocation", "tax", "guards"
	Logic        map[string]interface{} `json:"logic"`      // JsonLogic structure
	OutputKey    string                 `json:"output_key"` // Onde armazenar o resultado (Ex: order.AppliedTaxes.VAT)
	ErrorMessage string                 `json:"error_message,omitempty"`
}

type EngineResult struct {
	StateFragment map[string]interface{} `json:"stateFragment"`
	ServerDelta   bool                   `json:"serverDelta"`
	RulesVersion  string                 `json:"rulesVersion"`
	ExecutionLog  []ExecutionStep        `json:"executionLog"`
	GuardsHit     []GuardViolation       `json:"guardsHit"`
}

type ExecutionStep struct {
	Phase   string `json:"phase"`
	RuleID  string `json:"ruleId"`
	Action  string `json:"action"`
	Message string `json:"message"`
}

type GuardViolation struct {
	RuleID  string `json:"ruleId"`
	Reason  string `json:"reason"`
	Context string `json:"context"`
}

// --- Constantes e Erros ---
var (
	// Erro definido no domínio, mas acessível via interfaces
	ErrRuleExecutionFailed = fmt.Errorf("rule execution failed")
)
