package core

import (
	"context"
	"math/big"

	"github.com/pkg/errors"
	fastssz "github.com/prysmaticlabs/fastssz"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v4/config/features"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/crypto/hash"
	"github.com/prysmaticlabs/prysm/v4/crypto/rand"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

// eth1DataMajorityVote determines the appropriate eth1data for a block proposal using
// an algorithm called Voting with the Majority. The algorithm works as follows:
//   - Determine the timestamp for the start slot for the eth1 voting period.
//   - Determine the earliest and latest timestamps that a valid block can have.
//   - Determine the first block not before the earliest timestamp. This block is the lower bound.
//   - Determine the last block not after the latest timestamp. This block is the upper bound.
//   - If the last block is too early, use current eth1data from the beacon state.
//   - Filter out votes on unknown blocks and blocks which are outside of the range determined by the lower and upper bounds.
//   - If no blocks are left after filtering votes, use eth1data from the latest valid block.
//   - Otherwise:
//   - Determine the vote with the highest count. Prefer the vote with the highest eth1 block height in the event of a tie.
//   - This vote's block is the eth1 block to use for the block proposal.
func (s *Service) eth1DataMajorityVote(ctx context.Context, beaconState state.BeaconState) (*ethpb.Eth1Data, error) {
	ctx, cancel := context.WithTimeout(ctx, eth1dataTimeout)
	defer cancel()

	slot := beaconState.Slot()
	votingPeriodStartTime := s.slotStartTime(slot)

	if s.MockEth1Votes {
		return s.mockETH1DataVote(ctx, slot)
	}
	if !s.Eth1InfoFetcher.ExecutionClientConnected() {
		return s.randomETH1DataVote(ctx)
	}
	eth1DataNotification = false

	genesisTime, _ := s.Eth1InfoFetcher.GenesisExecutionChainInfo()
	followDistanceSeconds := params.BeaconConfig().Eth1FollowDistance * params.BeaconConfig().SecondsPerETH1Block
	latestValidTime := votingPeriodStartTime - followDistanceSeconds
	earliestValidTime := votingPeriodStartTime - 2*followDistanceSeconds

	// Special case for starting from a pre-mined genesis: the eth1 vote should be genesis until the chain has advanced
	// by ETH1_FOLLOW_DISTANCE. The head state should maintain the same ETH1Data until this condition has passed, so
	// trust the existing head for the right eth1 vote until we can get a meaningful value from the deposit contract.
	if latestValidTime < genesisTime+followDistanceSeconds {
		log.WithField("genesisTime", genesisTime).WithField("latestValidTime", latestValidTime).Warn("voting period before genesis + follow distance, using eth1data from head")
		return s.HeadFetcher.HeadETH1Data(), nil
	}

	lastBlockByLatestValidTime, err := s.Eth1BlockFetcher.BlockByTimestamp(ctx, latestValidTime)
	if err != nil {
		log.WithError(err).Error("Could not get last block by latest valid time")
		return s.randomETH1DataVote(ctx)
	}
	if lastBlockByLatestValidTime.Time < earliestValidTime {
		return s.HeadFetcher.HeadETH1Data(), nil
	}

	lastBlockDepositCount, lastBlockDepositRoot := s.DepositFetcher.DepositsNumberAndRootAtHeight(ctx, lastBlockByLatestValidTime.Number)
	if lastBlockDepositCount == 0 {
		return s.ChainStartFetcher.ChainStartEth1Data(), nil
	}

	if lastBlockDepositCount >= s.HeadFetcher.HeadETH1Data().DepositCount {
		h, err := s.Eth1BlockFetcher.BlockHashByHeight(ctx, lastBlockByLatestValidTime.Number)
		if err != nil {
			log.WithError(err).Error("Could not get hash of last block by latest valid time")
			return s.randomETH1DataVote(ctx)
		}
		return &ethpb.Eth1Data{
			BlockHash:    h.Bytes(),
			DepositCount: lastBlockDepositCount,
			DepositRoot:  lastBlockDepositRoot[:],
		}, nil
	}
	return s.HeadFetcher.HeadETH1Data(), nil
}

func (s *Service) slotStartTime(slot primitives.Slot) uint64 {
	startTime, _ := s.Eth1InfoFetcher.GenesisExecutionChainInfo()
	return slots.VotingPeriodStartTime(startTime, slot)
}

// canonicalEth1Data determines the canonical eth1data and eth1 block height to use for determining deposits.
func (s *Service) canonicalEth1Data(
	ctx context.Context,
	beaconState state.BeaconState,
	currentVote *ethpb.Eth1Data) (*ethpb.Eth1Data, *big.Int, error) {
	var eth1BlockHash [32]byte

	// Add in current vote, to get accurate vote tally
	if err := beaconState.AppendEth1DataVotes(currentVote); err != nil {
		return nil, nil, errors.Wrap(err, "could not append eth1 data votes to state")
	}
	hasSupport, err := blocks.Eth1DataHasEnoughSupport(beaconState, currentVote)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not determine if current eth1data vote has enough support")
	}
	var canonicalEth1Data *ethpb.Eth1Data
	if hasSupport {
		canonicalEth1Data = currentVote
		eth1BlockHash = bytesutil.ToBytes32(currentVote.BlockHash)
	} else {
		canonicalEth1Data = beaconState.Eth1Data()
		eth1BlockHash = bytesutil.ToBytes32(beaconState.Eth1Data().BlockHash)
	}
	if features.Get().DisableStakinContractCheck && eth1BlockHash == [32]byte{} {
		return canonicalEth1Data, new(big.Int).SetInt64(0), nil
	}
	_, canonicalEth1DataHeight, err := s.Eth1BlockFetcher.BlockExists(ctx, eth1BlockHash)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not fetch eth1data height")
	}
	return canonicalEth1Data, canonicalEth1DataHeight, nil
}

func (s *Service) mockETH1DataVote(ctx context.Context, slot primitives.Slot) (*ethpb.Eth1Data, error) {
	if !eth1DataNotification {
		log.Warn("Beacon Node is no longer connected to an ETH1 chain, so ETH1 data votes are now mocked.")
		eth1DataNotification = true
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
	slotInVotingPeriod := slot.ModSlot(params.BeaconConfig().SlotsPerEpoch.Mul(uint64(params.BeaconConfig().EpochsPerEth1VotingPeriod)))
	headState, err := s.HeadFetcher.HeadStateReadOnly(ctx)
	if err != nil {
		return nil, err
	}
	var enc []byte
	enc = fastssz.MarshalUint64(enc, uint64(slots.ToEpoch(slot))+uint64(slotInVotingPeriod))
	depRoot := hash.Hash(enc)
	blockHash := hash.Hash(depRoot[:])
	return &ethpb.Eth1Data{
		DepositRoot:  depRoot[:],
		DepositCount: headState.Eth1DepositIndex(),
		BlockHash:    blockHash[:],
	}, nil
}

func (s *Service) randomETH1DataVote(ctx context.Context) (*ethpb.Eth1Data, error) {
	if !eth1DataNotification {
		log.Warn("Beacon Node is no longer connected to an ETH1 chain, so ETH1 data votes are now random.")
		eth1DataNotification = true
	}
	headState, err := s.HeadFetcher.HeadStateReadOnly(ctx)
	if err != nil {
		return nil, err
	}

	// set random roots and block hashes to prevent a majority from being
	// built if the eth1 node is offline
	randGen := rand.NewGenerator()
	depRoot := hash.Hash(bytesutil.Bytes32(randGen.Uint64()))
	blockHash := hash.Hash(bytesutil.Bytes32(randGen.Uint64()))
	return &ethpb.Eth1Data{
		DepositRoot:  depRoot[:],
		DepositCount: headState.Eth1DepositIndex(),
		BlockHash:    blockHash[:],
	}, nil
}
