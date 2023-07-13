package p2p

import (
	p2ptypes "github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p/types"
	"reflect"

	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"google.golang.org/protobuf/proto"
)

// gossipTopicMappings represent the protocol ID to protobuf message type map for easy
// lookup.
var gossipTopicMappings = map[p2ptypes.GossipTopic]proto.Message{
	BlockSubnetTopicFormat:                    &ethpb.SignedBeaconBlock{},
	AttestationSubnetTopicFormat:              &ethpb.Attestation{},
	ExitSubnetTopicFormat:                     &ethpb.SignedVoluntaryExit{},
	ProposerSlashingSubnetTopicFormat:         &ethpb.ProposerSlashing{},
	AttesterSlashingSubnetTopicFormat:         &ethpb.AttesterSlashing{},
	AggregateAndProofSubnetTopicFormat:        &ethpb.SignedAggregateAttestationAndProof{},
	SyncContributionAndProofSubnetTopicFormat: &ethpb.SignedContributionAndProof{},
	SyncCommitteeSubnetTopicFormat:            &ethpb.SyncCommitteeMessage{},
	BlsToExecutionChangeSubnetTopicFormat:     &ethpb.SignedBLSToExecutionChange{},
}

// GossipTopicMappings is a function to return the assigned data type
// versioned by epoch.
func GossipTopicMappings(topic p2ptypes.GossipTopic, epoch primitives.Epoch) proto.Message {
	if topic == BlockSubnetTopicFormat {
		if epoch >= params.BeaconConfig().CapellaForkEpoch {
			return &ethpb.SignedBeaconBlockCapella{}
		}
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
func AllTopics() []p2ptypes.GossipTopic {
	var topics []p2ptypes.GossipTopic
	for k := range gossipTopicMappings {
		topics = append(topics, k)
	}
	return topics
}

// GossipTypeMapping is the inverse of GossipTopicMappings so that an arbitrary protobuf message
// can be mapped to a protocol ID string.
var GossipTypeMapping = make(map[reflect.Type]p2ptypes.GossipTopic, len(gossipTopicMappings))

func init() {
	for k, v := range gossipTopicMappings {
		GossipTypeMapping[reflect.TypeOf(v)] = k
	}
	// Specially handle Altair objects.
	GossipTypeMapping[reflect.TypeOf(&ethpb.SignedBeaconBlockAltair{})] = BlockSubnetTopicFormat
	// Specially handle Bellatrix objects.
	GossipTypeMapping[reflect.TypeOf(&ethpb.SignedBeaconBlockBellatrix{})] = BlockSubnetTopicFormat
	// Specially handle Capella objects
	GossipTypeMapping[reflect.TypeOf(&ethpb.SignedBeaconBlockCapella{})] = BlockSubnetTopicFormat
}
