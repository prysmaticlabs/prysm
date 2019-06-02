package hashutil

import (
	"github.com/prysmaticlabs/prysm/shared/ssz"
	"reflect"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// HashBeaconBlock hashes the full block without the proposer signature.
// The proposer signature is ignored in order obtain the same block hash used
// as the "block_root" property in the proposer signature data.
func HashBeaconBlock(bb *pb.BeaconBlock) ([32]byte, error) {
	if bb == nil || reflect.ValueOf(bb).IsNil() {
		return [32]byte{}, ErrNilProto
	}
	// Ignore the proposer signature by temporarily deleting it.
	sig := bb.Signature
	bb.Signature = nil
	defer func() { bb.Signature = sig }()

	return HashProto(bb)
}

// BlockSigningRoot uses Simple Serialize (SSZ) to determine the block header signing root
// given a beacon block. This is used as the parent root of subsequent blocks, for verifying
// headers, and also looking up blocks by this root in the DB.
func BlockSigningRoot(bb *pb.BeaconBlock) ([32]byte, error) {
	bodyRoot, err := ssz.TreeHash(bb)
	if err != nil {
        return [32]byte{}, err
	}
	header := &pb.BeaconBlockHeader{
		Slot: bb.Slot,
		ParentRoot: bb.ParentRoot,
		BodyRoot: bodyRoot[:],
	}
	return ssz.SigningRoot(header)
}