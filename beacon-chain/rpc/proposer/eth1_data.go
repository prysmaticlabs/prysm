package proposer

import (
	"context"
	"math/big"
	"math/rand"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// getEth1Data vote for a given slot.
//
// Selection for eth1data vote is as follows:
// - Determine the timestamp of the end of the voting period.
// - Determine the eth1 block for that timestamp.
// - Look back for ETH1_FOLLOW_DISTANCE'th ancestor from the above block.
// - Construct vote around that block.
//
// There have been some discussions around converging on eth1data votes faster by reading existing
// votes from the state and preferring the majority, even if this was not aligned with our view of
// ETH1 at the end of the voting period. This additional complexity has not been added here yet.
func (ps *Server) getEth1Data(ctx context.Context, slot uint64) (*ethpb.Eth1Data, error) {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.getEth1Data")
	defer span.End()

	// We cannot compute new eth1 votes in the early slots of ETH2.
	if slot < params.BeaconConfig().SlotsPerEth1VotingPeriod {
		return ps.ChainStartFetcher.ChainStartEth1Data(), nil
	}
	if ps.MockEth1Votes {
		log.Warn("INTEROP: Using mocked ETH1 data vote for block.")
		return ps.mockInteropETH1DataVote(ctx, slot)
	}
	if !ps.Eth1InfoFetcher.IsConnectedToETH1() {
		log.Warn("Not connected to eth1, serving random ETH1 data vote.")
		return ps.randomETH1DataVote(ctx)
	}

	// Determine the timestamp of the start of the voting period.
	headState, err := ps.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, err
	}
	votingPeriodStartSlot := slot - (slot % params.BeaconConfig().SlotsPerEth1VotingPeriod)
	timestamp := time.Unix(int64(headState.GenesisTime), 0 /*ns*/).Add(time.Duration(votingPeriodStartSlot*params.BeaconConfig().SecondsPerSlot) * time.Second)

	// Determine the eth1 block for that timestamp.
	blockNumber, err := ps.Eth1BlockFetcher.BlockNumberByTimestamp(ctx, uint64(timestamp.Unix()))
	if blockNumber == nil || err != nil {
		traceutil.AnnotateError(span, err)
		return nil, errors.Wrap(err, "could not get block number from timestamp")
	}
	// Use ETH1_FOLLOW_DISTANCE'th ancestor.
	blockNumber = new(big.Int).Sub(blockNumber, big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance)))

	// Construct a vote around that block.
	blockHash, err := ps.Eth1BlockFetcher.BlockHashByHeight(ctx, blockNumber)
	if err != nil {
		return nil, errors.Wrap(err, "could not fetch eth1 block")
	}
	depositCount, depositRoot := ps.DepositFetcher.DepositsNumberAndRootAtHeight(ctx, blockNumber)
	if depositCount == 0 {
		return ps.ChainStartFetcher.ChainStartEth1Data(), nil
	}

	// Sanity check: if our computed deposit count is less than headState.Eth1DepositIndex then
	// this proposal will fail processing.
	if depositCount < headState.Eth1DepositIndex {
		log.Warnf("Computed a lower deposit count (%d) than the eth1 deposit index in the "+
			"head state (%d). Using a random ETH1 data vote.", depositCount, headState.Eth1DepositIndex)
		return ps.randomETH1DataVote(ctx)
	}

	return &ethpb.Eth1Data{
		DepositRoot:  depositRoot[:],
		BlockHash:    blockHash[:],
		DepositCount: depositCount,
	}, nil
}

// If a mock eth1 data votes is specified, we use the following for the
// eth1data we provide to every proposer based on https://github.com/ethereum/eth2.0-pm/issues/62:
//
// slot_in_voting_period = current_slot % SLOTS_PER_ETH1_VOTING_PERIOD
// Eth1Data(
//   DepositRoot = hash(current_epoch + slot_in_voting_period),
//   DepositCount = state.eth1_deposit_index,
//   BlockHash = hash(hash(current_epoch + slot_in_voting_period)),
// )
func (ps *Server) mockInteropETH1DataVote(ctx context.Context, slot uint64) (*ethpb.Eth1Data, error) {
	slotInVotingPeriod := slot % params.BeaconConfig().SlotsPerEth1VotingPeriod
	headState, err := ps.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, err
	}
	enc, err := ssz.Marshal(helpers.SlotToEpoch(slot) + slotInVotingPeriod)
	if err != nil {
		return nil, err
	}
	depRoot := hashutil.Hash(enc)
	blockHash := hashutil.Hash(depRoot[:])
	return &ethpb.Eth1Data{
		DepositRoot:  depRoot[:],
		DepositCount: headState.Eth1DepositIndex,
		BlockHash:    blockHash[:],
	}, nil
}

func (ps *Server) randomETH1DataVote(ctx context.Context) (*ethpb.Eth1Data, error) {
	headState, err := ps.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, err
	}
	// set random roots and block hashes to prevent a majority from being
	// built if the eth1 node is offline
	depRoot := hashutil.Hash(bytesutil.Bytes32(rand.Uint64()))
	blockHash := hashutil.Hash(bytesutil.Bytes32(rand.Uint64()))
	return &ethpb.Eth1Data{
		DepositRoot:  depRoot[:],
		DepositCount: headState.Eth1DepositIndex,
		BlockHash:    blockHash[:],
	}, nil
}
