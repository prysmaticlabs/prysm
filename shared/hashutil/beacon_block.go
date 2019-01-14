package hashutil

import (
	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// HashBeaconBlock hashes the full block without the proposer signature.
// The proposer signature is ignored in order obtain the same block hash used
// as the "block_root" property in the proposer signature data.
func HashBeaconBlock(bb *pb.BeaconBlock) ([32]byte, error) {
	// Ignore the proposer signature by temporarily deleting it.
	sig := bb.Signature
	bb.Signature = nil
	defer func() { bb.Signature = sig }()

	// TODO(#1253): Use SSZ instead of proto marshal.
	data, err := proto.Marshal(bb)
	if err != nil {
		return [32]byte{}, err
	}

	return Hash(data), nil
}
