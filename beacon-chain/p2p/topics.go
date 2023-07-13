package p2p

import p2ptypes "github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p/types"

const (
	// GossipProtocolAndDigest represents the protocol and fork digest prefix in a gossip topic.
	GossipProtocolAndDigest = "/eth2/%x/"

	// Message Types
	//
	// GossipAttestationMessage is the name for the attestation message type. It is
	// specially extracted so as to determine the correct message type from an attestation
	// subnet.
	GossipAttestationMessage = "beacon_attestation"
	// GossipSyncCommitteeMessage is the name for the sync committee message type. It is
	// specially extracted so as to determine the correct message type from a sync committee
	// subnet.
	GossipSyncCommitteeMessage = "sync_committee"
	// GossipBlockMessage is the name for the block message type.
	GossipBlockMessage = "beacon_block"
	// GossipExitMessage is the name for the voluntary exit message type.
	GossipExitMessage = "voluntary_exit"
	// GossipProposerSlashingMessage is the name for the proposer slashing message type.
	GossipProposerSlashingMessage = "proposer_slashing"
	// GossipAttesterSlashingMessage is the name for the attester slashing message type.
	GossipAttesterSlashingMessage = "attester_slashing"
	// GossipAggregateAndProofMessage is the name for the attestation aggregate and proof message type.
	GossipAggregateAndProofMessage = "beacon_aggregate_and_proof"
	// GossipContributionAndProofMessage is the name for the sync contribution and proof message type.
	GossipContributionAndProofMessage = "sync_committee_contribution_and_proof"
	// GossipBlsToExecutionChangeMessage is the name for the bls to execution change message type.
	GossipBlsToExecutionChangeMessage = "bls_to_execution_change"
)

// Topic Formats
//
// AttestationSubnetTopicFormat is the topic format for the attestation subnet.
var AttestationSubnetTopicFormat = p2ptypes.GossipTopic{
	ProtocolPrefix: GossipProtocolAndDigest,
	BaseTopic:      GossipAttestationMessage + "_%d",
}

// SyncCommitteeSubnetTopicFormat is the topic format for the sync committee subnet.
var SyncCommitteeSubnetTopicFormat = p2ptypes.GossipTopic{
	ProtocolPrefix: GossipProtocolAndDigest,
	BaseTopic:      GossipSyncCommitteeMessage + "_%d",
}

// BlockSubnetTopicFormat is the topic format for the block subnet.
// var BlockSubnetTopicFormat = GossipProtocolAndDigest + GossipBlockMessage
var BlockSubnetTopicFormat = p2ptypes.GossipTopic{
	ProtocolPrefix: GossipProtocolAndDigest,
	BaseTopic:      GossipBlockMessage,
}

// ExitSubnetTopicFormat is the topic format for the voluntary exit subnet.
// var ExitSubnetTopicFormat = GossipProtocolAndDigest + GossipExitMessage
var ExitSubnetTopicFormat = p2ptypes.GossipTopic{
	ProtocolPrefix: GossipProtocolAndDigest,
	BaseTopic:      GossipExitMessage,
}

// ProposerSlashingSubnetTopicFormat is the topic format for the proposer slashing subnet.
// var ProposerSlashingSubnetTopicFormat = GossipProtocolAndDigest + GossipProposerSlashingMessage
var ProposerSlashingSubnetTopicFormat = p2ptypes.GossipTopic{
	ProtocolPrefix: GossipProtocolAndDigest,
	BaseTopic:      GossipProposerSlashingMessage,
}

// AttesterSlashingSubnetTopicFormat is the topic format for the attester slashing subnet.
// var AttesterSlashingSubnetTopicFormat = GossipProtocolAndDigest + GossipAttesterSlashingMessage
var AttesterSlashingSubnetTopicFormat = p2ptypes.GossipTopic{
	ProtocolPrefix: GossipProtocolAndDigest,
	BaseTopic:      GossipAttesterSlashingMessage,
}

// AggregateAndProofSubnetTopicFormat is the topic format for the aggregate and proof subnet.
// var AggregateAndProofSubnetTopicFormat = GossipProtocolAndDigest + GossipAggregateAndProofMessage
var AggregateAndProofSubnetTopicFormat = p2ptypes.GossipTopic{
	ProtocolPrefix: GossipProtocolAndDigest,
	BaseTopic:      GossipAggregateAndProofMessage,
}

// SyncContributionAndProofSubnetTopicFormat is the topic format for the sync aggregate and proof subnet.
// var SyncContributionAndProofSubnetTopicFormat = GossipProtocolAndDigest + GossipContributionAndProofMessage
var SyncContributionAndProofSubnetTopicFormat = p2ptypes.GossipTopic{
	ProtocolPrefix: GossipProtocolAndDigest,
	BaseTopic:      GossipContributionAndProofMessage,
}

// BlsToExecutionChangeSubnetTopicFormat is the topic format for the bls to execution change subnet.
// var BlsToExecutionChangeSubnetTopicFormat = GossipProtocolAndDigest + GossipBlsToExecutionChangeMessage
var BlsToExecutionChangeSubnetTopicFormat = p2ptypes.GossipTopic{
	ProtocolPrefix: GossipProtocolAndDigest,
	BaseTopic:      GossipBlsToExecutionChangeMessage,
}
