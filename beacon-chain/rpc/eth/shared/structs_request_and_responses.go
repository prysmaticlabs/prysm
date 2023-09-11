package shared

type GetStateForkResponse struct {
	Data                *Fork `json:"data"`
	ExecutionOptimistic bool  `json:"execution_optimistic"`
	Finalized           bool  `json:"finalized"`
}
