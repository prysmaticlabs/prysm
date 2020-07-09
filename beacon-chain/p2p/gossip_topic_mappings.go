package p2p

import (
	"reflect"

	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

// GossipTopicMappings represent the protocol ID to protobuf message type map for easy
// lookup.
var GossipTopicMappings = map[string]proto.Message{
	"/eth2/%x/beacon_block":               &pb.SignedBeaconBlock{},
	"/eth2/%x/beacon_attestation_%d":      &pb.Attestation{},
	"/eth2/%x/voluntary_exit":             &pb.SignedVoluntaryExit{},
	"/eth2/%x/proposer_slashing":          &pb.ProposerSlashing{},
	"/eth2/%x/attester_slashing":          &pb.AttesterSlashing{},
	"/eth2/%x/beacon_aggregate_and_proof": &pb.SignedAggregateAttestationAndProof{},
}

// GossipTypeMapping is the inverse of GossipTopicMappings so that an arbitrary protobuf message
// can be mapped to a protocol ID string.
var GossipTypeMapping = make(map[reflect.Type]string, len(GossipTopicMappings))

func init() {
	for k, v := range GossipTopicMappings {
		GossipTypeMapping[reflect.TypeOf(v)] = k
	}
}
