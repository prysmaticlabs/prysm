package blockutil

import (
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/ssz"
)

// BlockSigningRoot uses Simple Serialize (SSZ) to determine the block header signing root
// given a beacon block. This is used as the parent root of subsequent blocks, for verifying
// headers, and also looking up blocks by this root in the DB.
func BlockSigningRoot(bb *pb.BeaconBlock) ([32]byte, error) {
	bodyRoot, err := ssz.TreeHash(bb)
	if err != nil {
		return [32]byte{}, err
	}
	header := &pb.BeaconBlockHeader{
		Slot:       bb.Slot,
		ParentRoot: bb.ParentRoot,
		BodyRoot:   bodyRoot[:],
	}
	return ssz.SigningRoot(header)
}
