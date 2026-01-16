package engine

type EngineContext struct {
	Phase   string
	State   map[string]any
	Reasons []Reason
}
