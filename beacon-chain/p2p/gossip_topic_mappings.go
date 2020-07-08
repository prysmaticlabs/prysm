package p2p

import (
	"reflect"

	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

const (
	attestationSubnetTopicFormat       = "/eth2/%x/beacon_attestation_%d"
	blockSubnetTopicFormat             = "/eth2/%x/beacon_block"
	exitSubnetTopicFormat              = "/eth2/%x/voluntary_exit"
	proposerSlashingSubnetTopicFormat  = "/eth2/%x/proposer_slashing"
	attesterSlashingSubnetTopicFormat  = "/eth2/%x/attester_slashing"
	aggregateAndProofSubnetTopicFormat = "/eth2/%x/beacon_aggregate_and_proof"
)

// GossipTopicMappings represent the protocol ID to protobuf message type map for easy
// lookup.
var GossipTopicMappings = map[string]proto.Message{
	blockSubnetTopicFormat:             &pb.SignedBeaconBlock{},
	attestationSubnetTopicFormat:       &pb.Attestation{},
	exitSubnetTopicFormat:              &pb.SignedVoluntaryExit{},
	proposerSlashingSubnetTopicFormat:  &pb.ProposerSlashing{},
	attesterSlashingSubnetTopicFormat:  &pb.AttesterSlashing{},
	aggregateAndProofSubnetTopicFormat: &pb.SignedAggregateAttestationAndProof{},
}

// GossipTypeMapping is the inverse of GossipTopicMappings so that an arbitrary protobuf message
// can be mapped to a protocol ID string.
var GossipTypeMapping = make(map[reflect.Type]string)

func init() {
	for k, v := range GossipTopicMappings {
		GossipTypeMapping[reflect.TypeOf(v)] = k
	}
}
