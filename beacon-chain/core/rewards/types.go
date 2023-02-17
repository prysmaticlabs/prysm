package rewards

type Reward uint64

type ProposerRewards struct {
	Attestations      Reward
	AttesterSlashings Reward
	ProposerSlashings Reward
	SyncAggregate     Reward
}
