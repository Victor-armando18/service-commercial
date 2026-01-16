package engine

type Rule struct {
	ID    string
	Phase string
	Logic map[string]any
}
