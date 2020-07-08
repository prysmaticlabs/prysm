package p2p

import (
	"reflect"

	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

// GossipTopicMappings represent the protocol ID to protobuf message type map for easy
// lookup.
var GossipTopicMappings = map[string]proto.Message{
	BlockSubnetTopicFormat:             &pb.SignedBeaconBlock{},
	AttestationSubnetTopicFormat:       &pb.Attestation{},
	ExitSubnetTopicFormat:              &pb.SignedVoluntaryExit{},
	ProposerSlashingSubnetTopicFormat:  &pb.ProposerSlashing{},
	AttesterSlashingSubnetTopicFormat:  &pb.AttesterSlashing{},
	AggregateAndProofSubnetTopicFormat: &pb.SignedAggregateAttestationAndProof{},
}

// GossipTypeMapping is the inverse of GossipTopicMappings so that an arbitrary protobuf message
// can be mapped to a protocol ID string.
var GossipTypeMapping = make(map[reflect.Type]string)

func init() {
	for k, v := range GossipTopicMappings {
		GossipTypeMapping[reflect.TypeOf(v)] = k
	}
}
