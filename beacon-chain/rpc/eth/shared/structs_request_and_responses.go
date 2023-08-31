package shared

type StateForkResponse struct {
	Data                *Fork `json:"data"`
	ExecutionOptimistic bool  `json:"execution_optimistic"`
	Finalized           bool  `json:"finalized"`
}
