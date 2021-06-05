package p2p

const (
	// GossipAttestationMessage is the name for the attestation message type. It is
	// specially extracted so as to determine the correct message type from an attestation
	// subnet.
	GossipAttestationMessage = "beacon_attestation"
	// AttestationSubnetTopicFormat is the topic format for the attestation subnet.
	AttestationSubnetTopicFormat = "/eth2/%x/" + GossipAttestationMessage + "_%d"
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
