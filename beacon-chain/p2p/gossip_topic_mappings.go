package p2p

import (
	"reflect"

	types "github.com/prysmaticlabs/eth2-types"
	pb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/protobuf/proto"
)

// gossipTopicMappings represent the protocol ID to protobuf message type map for easy
// lookup.
var gossipTopicMappings = map[string]proto.Message{
	BlockSubnetTopicFormat:                    &pb.SignedBeaconBlock{},
	AttestationSubnetTopicFormat:              &pb.Attestation{},
	SyncCommitteeSubnetTopicFormat:            &pb.SyncCommitteeSignature{},
	ExitSubnetTopicFormat:                     &pb.SignedVoluntaryExit{},
	ProposerSlashingSubnetTopicFormat:         &pb.ProposerSlashing{},
	AttesterSlashingSubnetTopicFormat:         &pb.AttesterSlashing{},
	AggregateAndProofSubnetTopicFormat:        &pb.SignedAggregateAttestationAndProof{},
	SyncContributionAndProofSubnetTopicFormat: &pb.SignedContributionAndProof{},
}

func GossipTopicMappings(topic string, epoch types.Epoch) proto.Message {
	if topic == BlockSubnetTopicFormat && epoch >= params.BeaconConfig().AltairForkEpoch {
		return &pb.SignedBeaconBlockAltair{}
	}
	return gossipTopicMappings[topic]
}

func AllTopics() []string {
	topics := []string{}
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
	// Specially handle Altair Objects.
	GossipTypeMapping[reflect.TypeOf(&pb.SignedBeaconBlockAltair{})] = BlockSubnetTopicFormat
}
