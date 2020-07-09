package p2p

const (
	// AttestationSubnetTopicFormat is the topic format for the attestation subnet.
	AttestationSubnetTopicFormat = "/eth2/%x/beacon_attestation_%d"
	// BlockSubnetTopicFormat is the topic format for the block subnet.
	BlockSubnetTopicFormat = "/eth2/%x/beacon_block"
	// ExitSubnetTopicFormat is the topic format for the voluntary exit subnet.
	ExitSubnetTopicFormat = "/eth2/%x/voluntary_exit"
	// ProposerSlashingSubnetTopicFormat is the topic format for the proposer slashing subnet.
	ProposerSlashingSubnetTopicFormat = "/eth2/%x/proposer_slashing"
	// AttesterSlashingSubnetTopicFormat is the topic format for the attester slashing subnet.
	AttesterSlashingSubnetTopicFormat = "/eth2/%x/attester_slashing"
	// AggregateAndProofSubnetTopicFormat is the topic format for the aggregate and proof subnet.
	AggregateAndProofSubnetTopicFormat = "/eth2/%x/beacon_aggregate_and_proof"
)
