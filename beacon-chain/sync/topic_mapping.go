package sync

import (
	"reflect"

	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

// TODO: Move to appropriate place.

var GossipTopicMappings = map[string]proto.Message{
	"/eth2/beacon_block":       &pb.BeaconBlock{},
	"/eth2/beacon_attestation": &pb.Attestation{},
	"/eth2/voluntary_exit":     &pb.VoluntaryExit{},
	"/eth2/proposer_slashing":  &pb.ProposerSlashing{},
	"/eth2/attester_slashing":  &pb.AttesterSlashing{},
}

var GossipTypeMapping = make(map[reflect.Type]string)

func init() {
	for k, v := range GossipTopicMappings {
		GossipTypeMapping[reflect.TypeOf(v)] = k
	}
}
