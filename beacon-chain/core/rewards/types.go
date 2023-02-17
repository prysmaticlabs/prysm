package rewards

type ProposerReward uint64

type ProposerRewards struct {
	Attestations      ProposerReward
	AttesterSlashings ProposerReward
	ProposerSlashings ProposerReward
	SyncAggregate     ProposerReward
}
