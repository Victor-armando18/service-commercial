package engine

type Result struct {
	StateFragment map[string]any
	Delta         map[string]any
	Reasons       []Reason
	RulesVersion  string
}
