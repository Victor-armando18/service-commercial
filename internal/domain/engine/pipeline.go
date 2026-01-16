package engine

type PipelinePhase string

const (
	Baseline   PipelinePhase = "baseline"
	Allocation PipelinePhase = "allocation"
	Totals     PipelinePhase = "totals"
)
