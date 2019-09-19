package p2p

import (
	"reflect"

	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

// GossipTopicMappings represent the protocol ID to protobuf message type map for easy
// lookup.
var GossipTopicMappings = map[string]proto.Message{
	"/eth2/beacon_block":       &pb.BeaconBlock{},
	"/eth2/beacon_attestation": &pb.Attestation{},
	"/eth2/voluntary_exit":     &pb.VoluntaryExit{},
	"/eth2/proposer_slashing":  &pb.ProposerSlashing{},
	"/eth2/attester_slashing":  &pb.AttesterSlashing{},
}

// GossipTypeMapping is the inverse of GossipTopicMappings so that an arbitrary protobuf message
// can be mapped to a protocol ID string.
var GossipTypeMapping = make(map[reflect.Type]string)

func init() {
	for k, v := range GossipTopicMappings {
		GossipTypeMapping[reflect.TypeOf(v)] = k
	}
}
