// Package blocks contains block processing libraries. These libraries
// process and verify block specific messages such as PoW receipt root,
// RANDAO, validator deposits, exits and slashing proofs.
package blocks

import (
	"fmt"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var clock utils.Clock = &utils.RealClock{}

// NewGenesisBlock returns the canonical, genesis block for the beacon chain protocol.
func NewGenesisBlock(stateRoot []byte) *pb.BeaconBlock {
	zeroHash := params.BeaconConfig().ZeroHash[:]
	genBlock := &pb.BeaconBlock{
		ParentRoot: zeroHash,
		StateRoot:  stateRoot,
		Body:       &pb.BeaconBlockBody{},
		Signature:  params.BeaconConfig().EmptySignature[:],
	}
	return genBlock
}

// BlockFromHeader manufactures a block from its header. It contains all its fields,
// except for the block body.
func BlockFromHeader(header *pb.BeaconBlockHeader) *pb.BeaconBlock {
	return &pb.BeaconBlock{
		StateRoot:  header.StateRoot,
		Slot:       header.Slot,
		Signature:  header.Signature,
		ParentRoot: header.ParentRoot,
	}
}

// HeaderFromBlock extracts the block header from a block.
func HeaderFromBlock(block *pb.BeaconBlock) (*pb.BeaconBlockHeader, error) {
	header := &pb.BeaconBlockHeader{
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
