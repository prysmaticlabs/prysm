package hashutil_test

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

func TestHashBeaconBlock_doesntMutate(t *testing.T) {
	a := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Attestations: []*pb.Attestation{
				{
					Data: &pb.AttestationData{
						Slot:  123,
						Shard: 456,
					},
				},
			},
		},
		Signature: []byte{'S', 'I', 'G'},
	}
	b := proto.Clone(a).(*pb.BeaconBlock)

	_, err := hashutil.HashBeaconBlock(b)
	if err != nil {
		t.Error(err)
	}

	if !proto.Equal(a, b) {
		t.Error("Protos are not equal!")
	}
}
func TestHashProposal_doesntMutate(t *testing.T) {
	a := &pb.Proposal{
		Slot:            123,
		Shard:           456,
		BlockRootHash32: []byte{'R', 'O', 'O', 'T'},
		Signature:       []byte{'S', 'I', 'G'},
	}
	b := proto.Clone(a).(*pb.Proposal)

	_, err := hashutil.HashProposal(b)
	if err != nil {
		t.Error(err)
	}

	if !proto.Equal(a, b) {
		t.Error("Protos are not equal!")
	}
}
