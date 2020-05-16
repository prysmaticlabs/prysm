// Package testing includes useful helpers for slasher-related
// unit tests.
package testing

import (
	"crypto/rand"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// SignedBlockHeader given slot, proposer index this function generates signed block header.
// with random bytes as its signature.
func SignedBlockHeader(slot uint64, proposerIdx uint64) (*ethpb.SignedBeaconBlockHeader, error) {
	sig, err := genRandomSig()
	if err != nil {
		return nil, err
	}
	root := [32]byte{1, 2, 3}
	return &ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			ProposerIndex: proposerIdx,
			Slot:          slot,
			ParentRoot:    root[:],
			StateRoot:     root[:],
			BodyRoot:      root[:],
		},
		Signature: sig,
	}, nil
}

func genRandomSig() ([]byte, error) {
	blk := make([]byte, 96)
	_, err := rand.Read(blk)
	return blk, err
}

// StartSlot returns the first slot of a given epoch.
func StartSlot(epoch uint64) uint64 {
	return epoch * params.BeaconConfig().SlotsPerEpoch
}
