package engine

type RulePack struct {
	Version string
	Rules   []Rule
	Guards  []Guard
}
