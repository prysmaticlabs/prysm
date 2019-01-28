package rpc

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/bytes"
	ptypes"github.com/gogo/protobuf/types"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/gogo/protobuf/proto"
)

type ProposerServer struct {
	beaconDB *db.BeaconDB
	chainService          chainService
	powChainService    powChainService
	canonicalStateChan    chan *pbp2p.BeaconState
	enablePOWChain bool
}

// ProposeBlock is called by a proposer in a sharding validator and a full beacon node
// sends the request into a beacon block that can then be included in a canonical chain.
func (ps *ProposerServer) ProposeBlock(ctx context.Context, blk *pbp2p.BeaconBlock) (*pb.ProposeResponse, error) {
	h, err := hashutil.HashBeaconBlock(blk)
	if err != nil {
		return nil, fmt.Errorf("could not hash block: %v", err)
	}
	log.WithField("blockHash", fmt.Sprintf("%#x", h)).Debugf("Block proposal received via RPC")
	// We relay the received block from the proposer to the chain service for processing.
	ps.chainService.IncomingBlockFeed().Send(blk)
	return &pb.ProposeResponse{BlockHash: h[:]}, nil
}

// LatestPOWChainBlockHash retrieves the latest blockhash of the POW chain and sends it to the validator client.
func (ps *ProposerServer) LatestPOWChainBlockHash(ctx context.Context, req *ptypes.Empty) (*pb.POWChainResponse, error) {
	var powChainHash common.Hash
	if !ps.enablePOWChain {
		powChainHash = common.BytesToHash([]byte{'p', 'o', 'w', 'c', 'h', 'a', 'i', 'n'})

		return &pb.POWChainResponse{
			BlockHash: powChainHash[:],
		}, nil
	}

	powChainHash = ps.powChainService.LatestBlockHash()
	return &pb.POWChainResponse{
		BlockHash: powChainHash[:],
	}, nil
}

// ComputeStateRoot computes the state root after a block has been processed through a state transition and
// returns it to the validator client.
func (ps *ProposerServer) ComputeStateRoot(ctx context.Context, req *pbp2p.BeaconBlock) (*pb.StateRootResponse, error) {
	beaconState, err := ps.beaconDB.State()
	if err != nil {
		return nil, fmt.Errorf("could not get beacon state: %v", err)
	}

	parentHash := bytes.ToBytes32(req.ParentRootHash32)
	beaconState, err = state.ExecuteStateTransition(beaconState, req, parentHash)
	if err != nil {
		return nil, fmt.Errorf("could not execute state transition %v", err)
	}

	encodedState, err := proto.Marshal(beaconState)
	if err != nil {
		return nil, fmt.Errorf("could not marshal state %v", err)
	}

	beaconStateHash := hashutil.Hash(encodedState)
	log.WithField("beaconStateHash", fmt.Sprintf("%#x", beaconStateHash)).Debugf("Computed state hash")
	return &pb.StateRootResponse{
		StateRoot: beaconStateHash[:],
	}, nil
}

// ProposerIndex sends a response to the client which returns the proposer index for a given slot. Validators
// are shuffled and assigned slots to attest/propose to. This method will look for the validator that is assigned
// to propose a beacon block at the given slot.
func (ps *ProposerServer) ProposerIndex(ctx context.Context, req *pb.ProposerIndexRequest) (*pb.IndexResponse, error) {
	beaconState, err := ps.beaconDB.State()
	if err != nil {
		return nil, fmt.Errorf("could not get beacon state: %v", err)
	}

	_, ProposerIndex, err := v.ProposerShardAndIdx(
		beaconState,
		req.SlotNumber,
	)
	if err != nil {
		return nil, fmt.Errorf("could not get index of previous proposer: %v", err)
	}

	return &pb.IndexResponse{
		Index: uint32(ProposerIndex),
	}, nil
}
