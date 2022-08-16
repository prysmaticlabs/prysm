package p2p

import (
	"reflect"

	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"google.golang.org/protobuf/proto"
)

// gossipTopicMappings represent the protocol ID to protobuf message type map for easy
// lookup.
var gossipTopicMappings = map[string]proto.Message{
	BlockSubnetTopicFormat:                    &ethpb.SignedBeaconBlock{},
	AttestationSubnetTopicFormat:              &ethpb.Attestation{},
	ExitSubnetTopicFormat:                     &ethpb.SignedVoluntaryExit{},
	ProposerSlashingSubnetTopicFormat:         &ethpb.ProposerSlashing{},
	AttesterSlashingSubnetTopicFormat:         &ethpb.AttesterSlashing{},
	AggregateAndProofSubnetTopicFormat:        &ethpb.SignedAggregateAttestationAndProof{},
	SyncContributionAndProofSubnetTopicFormat: &ethpb.SignedContributionAndProof{},
	SyncCommitteeSubnetTopicFormat:            &ethpb.SyncCommitteeMessage{},
}

// GossipTopicMappings is a function to return the assigned data type
// versioned by epoch.
func GossipTopicMappings(topic string, epoch types.Epoch) proto.Message {
	if topic == BlockSubnetTopicFormat {
		if epoch >= params.BeaconConfig().BellatrixForkEpoch {
			return &ethpb.SignedBeaconBlockBellatrix{}
		}
		if epoch >= params.BeaconConfig().AltairForkEpoch {
			return &ethpb.SignedBeaconBlockAltair{}
		}
	}
	return gossipTopicMappings[topic]
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
		GossipTypeMapping[reflect.TypeOf(v)] = k
	}
	// Specially handle Altair objects.
	GossipTypeMapping[reflect.TypeOf(&ethpb.SignedBeaconBlockAltair{})] = BlockSubnetTopicFormat
	// Specially handle Bellatrix objects.
	GossipTypeMapping[reflect.TypeOf(&ethpb.SignedBeaconBlockBellatrix{})] = BlockSubnetTopicFormat
	GossipTypeMapping[reflect.TypeOf(&ethpb.SignedBlindedBeaconBlockBellatrix{})] = BlockSubnetTopicFormat
}
