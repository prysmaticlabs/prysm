// Package blocks contains block processing libraries. These libraries
// process and verify block specific messages such as PoW receipt root,
// RANDAO, validator deposits, exits and slashing proofs.
package blocks

import (
	"fmt"

	"github.com/prysmaticlabs/go-ssz"
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

// BlockFromHeader manufactures a block from its header. It contains all its fields,
// except for the block body.
func BlockFromHeader(header *ethpb.BeaconBlockHeader) *ethpb.BeaconBlock {
	return &ethpb.BeaconBlock{
		StateRoot:  header.StateRoot,
		Slot:       header.Slot,
		Signature:  header.Signature,
		ParentRoot: header.ParentRoot,
	}
}

// HeaderFromBlock extracts the block header from a block.
func HeaderFromBlock(block *ethpb.BeaconBlock) (*ethpb.BeaconBlockHeader, error) {
	header := &ethpb.BeaconBlockHeader{
		Slot:       block.Slot,
		ParentRoot: block.ParentRoot,
		Signature:  block.Signature,
		StateRoot:  block.StateRoot,
	}
	root, err := ssz.HashTreeRoot(block.Body)
	if err != nil {
		return nil, fmt.Errorf("could not tree hash block body %v", err)
	}
	header.BodyRoot = root[:]
	return header, nil
}
