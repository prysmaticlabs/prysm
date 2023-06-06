package rewards

type BlockRewardsResponse struct {
	Data                BlockRewards `json:"data"`
	ExecutionOptimistic bool         `json:"execution_optimistic"`
	Finalized           bool         `json:"finalized"`
}

type BlockRewards struct {
	ProposerIndex     string `json:"proposer_index"`
	Total             string `json:"total"`
	Attestations      string `json:"attestations"`
	SyncAggregate     string `json:"sync_aggregate"`
	ProposerSlashings string `json:"proposer_slashings"`
	AttesterSlashings string `json:"attester_slashings"`
}

type AttestationRewardsResponse struct {
	Data                AttestationRewards `json:"data"`
	ExecutionOptimistic bool               `json:"execution_optimistic"`
	Finalized           bool               `json:"finalized"`
}

type AttestationRewards struct {
	IdealRewards []IdealAttestationReward
	TotalRewards []TotalAttestationReward
}

type AttReward interface {
	SetHead(value string)
	GetTarget() string
	SetTarget(value string)
	SetSource(value string)
}

type IdealAttestationReward struct {
	EffectiveBalance string `json:"effective_balance"`
	Head             string `json:"head"`
	Target           string `json:"target"`
	Source           string `json:"source"`
}

func (r *IdealAttestationReward) SetHead(value string) {
	r.Head = value
}

func (r *IdealAttestationReward) GetTarget() string {
	return r.Target
}

func (r *IdealAttestationReward) SetTarget(value string) {
	r.Target = value
}

func (r *IdealAttestationReward) SetSource(value string) {
	r.Source = value
}

type TotalAttestationReward struct {
	ValidatorIndex string `json:"validator_index"`
	Head           string `json:"head"`
	Target         string `json:"target"`
	Source         string `json:"source"`
	InclusionDelay string `json:"inclusion_delay"`
}

func (r *TotalAttestationReward) SetHead(value string) {
	r.Head = value
}

func (r *TotalAttestationReward) GetTarget() string {
	return r.Target
}

func (r *TotalAttestationReward) SetTarget(value string) {
	r.Target = value
}

func (r *TotalAttestationReward) SetSource(value string) {
	r.Source = value
}
