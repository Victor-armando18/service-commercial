package engine

import "context"

type OrderItem struct {
	SKU   string  `json:"sku"`
	Value float64 `json:"value"`
	Qty   int     `json:"qty"`
}

type Order struct {
	ID                 string             `json:"id"`
	Currency           string             `json:"currency"`
	CorrelationId      string             `json:"correlationId"`
	RulesVersion       string             `json:"rulesVersion"`
	Items              []OrderItem        `json:"items"`
	BaseValue          float64            `json:"baseValue"`
	TotalValue         float64            `json:"totalValue"`
	TotalItems         int                `json:"totalItems"`
	DiscountPercentage float64            `json:"discountPercentage"`
	AppliedTaxes       map[string]float64 `json:"appliedTaxes"`
}

type RuleConfig struct {
	ID           string                 `json:"id"`
	Phase        string                 `json:"phase"`
	Logic        map[string]interface{} `json:"logic"`
	OutputKey    string                 `json:"output_key"`
	ErrorMessage string                 `json:"error_message"`
}

type RulePack struct {
	Version     string       `json:"version"`
	Description string       `json:"description"`
	Rules       []RuleConfig `json:"rules"`
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

type EngineResult struct {
	StateFragment map[string]interface{} `json:"stateFragment"`
	ServerDelta   bool                   `json:"serverDelta"`
	RulesVersion  string                 `json:"rulesVersion"`
	ExecutionLog  []ExecutionStep        `json:"executionLog"`
	GuardsHit     []GuardViolation       `json:"guardsHit"`
}

type RulePackLoader interface {
	Load(ctx context.Context, version string) (*RulePack, error)
}
