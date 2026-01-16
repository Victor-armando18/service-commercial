package engine

type Guard struct {
	ID     string
	Reason string
	Logic  map[string]any
}
