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
	"github.com/prysmaticlabs/prysm/shared/ssz"
)

var clock utils.Clock = &utils.RealClock{}

// NewGenesisBlock returns the canonical, genesis block for the beacon chain protocol.
func NewGenesisBlock(stateRoot []byte) *pb.BeaconBlock {
	block := &pb.BeaconBlock{
		Slot:       0,
		ParentRoot: params.BeaconConfig().ZeroHash[:],
		StateRoot:  stateRoot,
		Signature:  params.BeaconConfig().EmptySignature[:],
		Body: &pb.BeaconBlockBody{
			RandaoReveal:      params.BeaconConfig().ZeroHash[:],
			ProposerSlashings: []*pb.ProposerSlashing{},
			AttesterSlashings: []*pb.AttesterSlashing{},
			Attestations:      []*pb.Attestation{},
			Deposits:          []*pb.Deposit{},
			VoluntaryExits:    []*pb.VoluntaryExit{},
			Eth1Data: &pb.Eth1Data{
				DepositRoot: params.BeaconConfig().ZeroHash[:],
				BlockRoot:   params.BeaconConfig().ZeroHash[:],
			},
		},
	}
	return block
}

// BlockFromHeader manufactures a block from its header. It contains all its fields,
// expect for the block body.
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
	root, err := ssz.TreeHash(block.Body)
	if err != nil {
		return nil, fmt.Errorf("could not tree hash block body %v", err)
	}
	header.BodyRoot = root[:]
	return header, nil
}
