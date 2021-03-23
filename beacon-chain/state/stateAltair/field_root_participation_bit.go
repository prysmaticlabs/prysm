package stateAltair

import "github.com/prysmaticlabs/prysm/beacon-chain/p2p/types"

// participationBitsRoot computes the HashTreeRoot merkleization of
// participation roots.
func participationBitsRoot(bits []byte) ([32]byte, error) {
	bitsSSZ := types.SSZBytes(bits)
	bitsSSZHTR, err := bitsSSZ.HashTreeRoot()
	if err != nil {
		return [32]byte{}, err
	}
	return bitsSSZHTR, nil
}
