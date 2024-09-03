package p2p

import (
	"reflect"

	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"google.golang.org/protobuf/proto"
)

// gossipTopicMappings represent the protocol ID to protobuf message type map for easy
// lookup.
var gossipTopicMappings = map[string]func() proto.Message{
	BeaconBlockSubnetTopicFormat:                       func() proto.Message { return &ethpb.SignedBeaconBlock{} },
	BeaconAttestationSubnetTopicFormat:                 func() proto.Message { return &ethpb.Attestation{} },
	VoluntaryExitSubnetTopicFormat:                     func() proto.Message { return &ethpb.SignedVoluntaryExit{} },
	ProposerSlashingSubnetTopicFormat:                  func() proto.Message { return &ethpb.ProposerSlashing{} },
	AttesterSlashingSubnetTopicFormat:                  func() proto.Message { return &ethpb.AttesterSlashing{} },
	BeaconAggregateAndProofSubnetTopicFormat:           func() proto.Message { return &ethpb.SignedAggregateAttestationAndProof{} },
	SyncCommitteeContributionAndProofSubnetTopicFormat: func() proto.Message { return &ethpb.SignedContributionAndProof{} },
	SyncCommitteeSubnetTopicFormat:                     func() proto.Message { return &ethpb.SyncCommitteeMessage{} },
	BlsToExecutionChangeSubnetTopicFormat:              func() proto.Message { return &ethpb.SignedBLSToExecutionChange{} },
	BlobSubnetTopicFormat:                              func() proto.Message { return &ethpb.BlobSidecar{} },
	DataColumnSubnetTopicFormat:                        func() proto.Message { return &ethpb.DataColumnSidecar{} },
}

// GossipTopicMappings is a function to return the assigned data type
// versioned by epoch.
func GossipTopicMappings(topic string, epoch primitives.Epoch) proto.Message {
	switch topic {
	case BeaconBlockSubnetTopicFormat:
		if epoch >= params.BeaconConfig().ElectraForkEpoch {
			return &ethpb.SignedBeaconBlockElectra{}
		}
		if epoch >= params.BeaconConfig().DenebForkEpoch {
			return &ethpb.SignedBeaconBlockDeneb{}
		}
		if epoch >= params.BeaconConfig().CapellaForkEpoch {
			return &ethpb.SignedBeaconBlockCapella{}
		}
		if epoch >= params.BeaconConfig().BellatrixForkEpoch {
			return &ethpb.SignedBeaconBlockBellatrix{}
		}
		if epoch >= params.BeaconConfig().AltairForkEpoch {
			return &ethpb.SignedBeaconBlockAltair{}
		}
		return gossipMessage(topic)
	case BeaconAttestationSubnetTopicFormat:
		if epoch >= params.BeaconConfig().ElectraForkEpoch {
			return &ethpb.AttestationElectra{}
		}
		return gossipMessage(topic)
	case AttesterSlashingSubnetTopicFormat:
		if epoch >= params.BeaconConfig().ElectraForkEpoch {
			return &ethpb.AttesterSlashingElectra{}
		}
		return gossipMessage(topic)
	case BeaconAggregateAndProofSubnetTopicFormat:
		if epoch >= params.BeaconConfig().ElectraForkEpoch {
			return &ethpb.SignedAggregateAttestationAndProofElectra{}
		}
		return gossipMessage(topic)
	default:
		return gossipMessage(topic)
	}
}

func gossipMessage(topic string) proto.Message {
	msgGen, ok := gossipTopicMappings[topic]
	if !ok {
		return nil
	}
	return msgGen()
}

// AllTopics returns all topics stored in our
// gossip mapping.
func AllTopics() []string {
	var topics []string
	for k := range gossipTopicMappings {
		topics = append(topics, k)
	}
	return topics
}

// GossipTypeMapping is the inverse of GossipTopicMappings so that an arbitrary protobuf message
// can be mapped to a protocol ID string.
var GossipTypeMapping = make(map[reflect.Type]string, len(gossipTopicMappings))

func init() {
	for k, v := range gossipTopicMappings {
		GossipTypeMapping[reflect.TypeOf(v())] = k
	}
	// Specially handle Altair objects.
	GossipTypeMapping[reflect.TypeOf(&ethpb.SignedBeaconBlockAltair{})] = BeaconBlockSubnetTopicFormat
	// Specially handle Bellatrix objects.
	GossipTypeMapping[reflect.TypeOf(&ethpb.SignedBeaconBlockBellatrix{})] = BeaconBlockSubnetTopicFormat
	// Specially handle Capella objects.
	GossipTypeMapping[reflect.TypeOf(&ethpb.SignedBeaconBlockCapella{})] = BeaconBlockSubnetTopicFormat
	// Specially handle Deneb objects.
	GossipTypeMapping[reflect.TypeOf(&ethpb.SignedBeaconBlockDeneb{})] = BeaconBlockSubnetTopicFormat
	// Specially handle Electra objects.
	GossipTypeMapping[reflect.TypeOf(&ethpb.SignedBeaconBlockElectra{})] = BeaconBlockSubnetTopicFormat
	GossipTypeMapping[reflect.TypeOf(&ethpb.AttestationElectra{})] = BeaconAttestationSubnetTopicFormat
	GossipTypeMapping[reflect.TypeOf(&ethpb.AttesterSlashingElectra{})] = AttesterSlashingSubnetTopicFormat
	GossipTypeMapping[reflect.TypeOf(&ethpb.SignedAggregateAttestationAndProofElectra{})] = BeaconAggregateAndProofSubnetTopicFormat
}
