package rewards

type BlockRewardsResponse struct {
	Data                *BlockRewards `json:"data"`
	ExecutionOptimistic bool          `json:"execution_optimistic"`
	Finalized           bool          `json:"finalized"`
}

type BlockRewards struct {
	ProposerIndex     string `json:"proposer_index"`
	Total             string `json:"total"`
	Attestations      string `json:"attestations"`
	SyncAggregate     string `json:"sync_aggregate"`
	ProposerSlashings string `json:"proposer_slashings"`
	AttesterSlashings string `json:"attester_slashings"`
}
