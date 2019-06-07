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

func TestHashBeaconBlock_nil(t *testing.T) {
	var blk *pb.BeaconBlock
	_, err := hashutil.HashBeaconBlock(blk)
	if err != hashutil.ErrNilProto {
		t.Fatalf("Error from hashing nil block is not the correct type, instead it is: %v", err)
	}
}
