// Package blocks contains block processing libraries. These libraries
// process and verify block specific messages such as PoW receipt root,
// RANDAO, validator deposits, exits and slashing proofs.
package blocks

import (
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// NewGenesisBlock returns the canonical, genesis block for the beacon chain protocol.
func NewGenesisBlock(stateRoot []byte) *ethpb.BeaconBlock {
	zeroHash := params.BeaconConfig().ZeroHash[:]
	genBlock := &ethpb.BeaconBlock{
		ParentRoot: zeroHash,
		StateRoot:  stateRoot,
		Body:       &ethpb.BeaconBlockBody{},
		Signature:  params.BeaconConfig().EmptySignature[:],
	}
	return genBlock
}
