// Package blocks contains block processing libraries. These libraries
// process and verify block specific messages such as PoW receipt root,
// RANDAO, validator deposits, exits and slashing proofs.
package blocks

import (
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// NewGenesisBlock returns the canonical, genesis block for the beacon chain protocol.
func NewGenesisBlock(stateRoot []byte) *ethpb.SignedBeaconBlock {
	zeroHash := params.BeaconConfig().ZeroHash[:]
	genBlock := &ethpb.BeaconBlock{
		ParentRoot: zeroHash,
		StateRoot:  stateRoot,
		Body:       &ethpb.BeaconBlockBody{},
	}
	return &ethpb.SignedBeaconBlock{
		Block:     genBlock,
		Signature: params.BeaconConfig().EmptySignature[:],
	}
}
