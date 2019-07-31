package rpc

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	"github.com/sirupsen/logrus"
)

// ProposerServer defines a server implementation of the gRPC Proposer service,
// providing RPC endpoints for computing state transitions and state roots, proposing
// beacon blocks to a beacon node, and more.
type ProposerServer struct {
	beaconDB           *db.BeaconDB
	chainService       chainService
	powChainService    powChainService
	operationService   operationService
	canonicalStateChan chan *pbp2p.BeaconState
}

// RequestBlock is called by a proposer during its assigned slot to request a block to sign
// by passing in the slot and the signed randao reveal of the slot.
func (ps *ProposerServer) RequestBlock(ctx context.Context, req *pb.BlockRequest) (*ethpb.BeaconBlock, error) {

	// Retrieve the parent block as the current head of the canonical chain
	parent, err := ps.beaconDB.ChainHead()
	if err != nil {
		return nil, fmt.Errorf("could not get canonical head block: %v", err)
	}

	parentRoot, err := ssz.SigningRoot(parent)
	if err != nil {
		return nil, fmt.Errorf("could not get parent block signing root: %v", err)
	}

	// Construct block body
	// Pack ETH1 deposits which have not been included in the beacon chain
	eth1Data, err := ps.eth1Data(ctx, req.Slot)
	if err != nil {
		return nil, fmt.Errorf("could not get ETH1 data: %v", err)
	}

	// Pack ETH1 deposits which have not been included in the beacon chain.
	deposits, err := ps.deposits(ctx, eth1Data)
	if err != nil {
		return nil, fmt.Errorf("could not get eth1 deposits: %v", err)
	}

	// Pack aggregated attestations which have not been included in the beacon chain.
	attestations, err := ps.attestations(ctx, req.Slot)
	if err != nil {
		return nil, fmt.Errorf("could not get pending attestations: %v", err)
	}

	// Use zero hash as stub for state root to compute later.
	stateRoot := params.BeaconConfig().ZeroHash[:]

	emptySig := make([]byte, 96)

	blk := &ethpb.BeaconBlock{
		Slot:       req.Slot,
		ParentRoot: parentRoot[:],
		StateRoot:  stateRoot,
		Body: &ethpb.BeaconBlockBody{
			Eth1Data:     eth1Data,
			Deposits:     deposits,
			Attestations: attestations,
			RandaoReveal: req.RandaoReveal,
			// TODO(2766): Implement rest of the retrievals for beacon block operations
			Transfers:         []*ethpb.Transfer{},
			ProposerSlashings: []*ethpb.ProposerSlashing{},
			AttesterSlashings: []*ethpb.AttesterSlashing{},
			VoluntaryExits:    []*ethpb.VoluntaryExit{},
			Graffiti:          []byte{},
		},
		Signature: emptySig,
	}

	// Compute state root with the newly constructed block.
	stateRoot, err = ps.computeStateRoot(ctx, blk)
	if err != nil {
		return nil, fmt.Errorf("could not get compute state root: %v", err)
	}
	blk.StateRoot = stateRoot

	return blk, nil
}

// ProposeBlock is called by a proposer during its assigned slot to create a block in an attempt
// to get it processed by the beacon node as the canonical head.
func (ps *ProposerServer) ProposeBlock(ctx context.Context, blk *ethpb.BeaconBlock) (*pb.ProposeResponse, error) {
	root, err := ssz.SigningRoot(blk)
	if err != nil {
		return nil, fmt.Errorf("could not tree hash block: %v", err)
	}
	log.WithField("blockRoot", fmt.Sprintf("%#x", bytesutil.Trunc(root[:]))).Debugf(
		"Block proposal received via RPC")

	beaconState, err := ps.chainService.ReceiveBlock(ctx, blk)
	if err != nil {
		return nil, fmt.Errorf("could not process beacon block: %v", err)
	}

	if err := ps.beaconDB.UpdateChainHead(ctx, blk, beaconState); err != nil {
		return nil, fmt.Errorf("failed to update chain: %v", err)
	}

	ps.chainService.UpdateCanonicalRoots(blk, root)
	log.WithFields(logrus.Fields{
		"headRoot": fmt.Sprintf("%#x", bytesutil.Trunc(root[:])),
		"headSlot": blk.Slot,
	}).Info("Chain head block and state updated")

	return &pb.ProposeResponse{BlockRoot: root[:]}, nil
}

// attestations retrieves aggregated attestations kept in the beacon node's operations pool which have
// not yet been included into the beacon chain. Proposers include these pending attestations in their
// proposed blocks when performing their responsibility. If desired, callers can choose to filter pending
// attestations which are ready for inclusion. That is, attestations that satisfy:
// attestation.slot + MIN_ATTESTATION_INCLUSION_DELAY <= state.slot.
func (ps *ProposerServer) attestations(ctx context.Context, expectedSlot uint64) ([]*ethpb.Attestation, error) {
	beaconState, err := ps.beaconDB.HeadState(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve beacon state: %v", err)
	}
	atts, err := ps.operationService.AttestationPool(ctx, expectedSlot)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve pending attestations from operations service: %v", err)
	}

	// advance slot, if it is behind
	if beaconState.Slot < expectedSlot {
		beaconState, err = state.ProcessSlots(ctx, beaconState, expectedSlot)
		if err != nil {
			return nil, err
		}
	}

	var attsReadyForInclusion []*ethpb.Attestation
	for _, att := range atts {
		slot, err := helpers.AttestationDataSlot(beaconState, att.Data)
		if err != nil {
			return nil, fmt.Errorf("could not get attestation slot: %v", err)
		}
		if slot+params.BeaconConfig().MinAttestationInclusionDelay <= beaconState.Slot &&
			beaconState.Slot <= slot+params.BeaconConfig().SlotsPerEpoch {
			attsReadyForInclusion = append(attsReadyForInclusion, att)
		}
	}

	validAtts := make([]*ethpb.Attestation, 0, len(attsReadyForInclusion))
	for _, att := range attsReadyForInclusion {
		slot, err := helpers.AttestationDataSlot(beaconState, att.Data)
		if err != nil {
			return nil, fmt.Errorf("could not get attestation slot: %v", err)
		}

		if _, err := blocks.ProcessAttestation(beaconState, att); err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}

			log.WithError(err).WithFields(logrus.Fields{
				"slot":     slot,
				"headRoot": fmt.Sprintf("%#x", bytesutil.Trunc(att.Data.BeaconBlockRoot))}).Info(
				"Deleting failed pending attestation from DB")
			if err := ps.beaconDB.DeleteAttestation(att); err != nil {
				return nil, fmt.Errorf("could not delete failed attestation: %v", err)
			}
			continue
		}
		canonical, err := ps.operationService.IsAttCanonical(ctx, att)
		if err != nil {
			// Delete attestation that failed to verify as canonical.
			if err := ps.beaconDB.DeleteAttestation(att); err != nil {
				return nil, fmt.Errorf("could not delete failed attestation: %v", err)
			}
			return nil, fmt.Errorf("could not verify canonical attestation: %v", err)
		}
		// Skip the attestation if it's not canonical.
		if !canonical {
			continue
		}

		validAtts = append(validAtts, att)
	}

	return validAtts, nil
}

// eth1Data determines the appropriate eth1data for a block proposal. The algorithm for this method
// is as follows:
//  - Determine the timestamp for the start slot for the eth1 voting period.
//  - Determine the most recent eth1 block before that timestamp.
//  - Subtract that eth1block.number by ETH1_FOLLOW_DISTANCE.
//  - This is the eth1block to use for the block proposal.
func (ps *ProposerServer) eth1Data(ctx context.Context, slot uint64) (*ethpb.Eth1Data, error) {
	eth1VotingPeriodStartTime := ps.powChainService.ETH2GenesisTime()
	eth1VotingPeriodStartTime += (slot - (slot % params.BeaconConfig().SlotsPerEth1VotingPeriod)) * params.BeaconConfig().SecondsPerSlot

	// Look up most recent block up to timestamp
	blockNumber, err := ps.powChainService.BlockNumberByTimestamp(ctx, eth1VotingPeriodStartTime)
	if err != nil {
		return nil, err
	}

	return ps.defaultEth1DataResponse(ctx, blockNumber)
}

// computeStateRoot computes the state root after a block has been processed through a state transition and
// returns it to the validator client.
func (ps *ProposerServer) computeStateRoot(ctx context.Context, block *ethpb.BeaconBlock) ([]byte, error) {
	beaconState, err := ps.beaconDB.HeadState(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get beacon state: %v", err)
	}
	s, err := state.ExecuteStateTransitionNoVerify(
		ctx,
		beaconState,
		block,
	)
	if err != nil {
		return nil, fmt.Errorf("could not execute state transition for state: %v at slot %d", err, beaconState.Slot)
	}

	root, err := ssz.HashTreeRoot(s)
	if err != nil {
		return nil, fmt.Errorf("could not tree hash beacon state: %v", err)
	}
	log.WithField("beaconStateRoot", fmt.Sprintf("%#x", root)).Debugf("Computed state hash")
	return root[:], nil
}

// deposits returns a list of pending deposits that are ready for inclusion in the next beacon
// block. Determining deposits depends on the current eth1data vote for the block and whether or not
// this eth1data has enough support to be considered for deposits inclusion. If current vote has
// enough support, then use that vote for basis of determining deposits, otherwise use current state
// eth1data.
func (ps *ProposerServer) deposits(ctx context.Context, currentVote *ethpb.Eth1Data) ([]*ethpb.Deposit, error) {
	bNum := ps.powChainService.LatestBlockHeight()
	if bNum == nil {
		return nil, errors.New("latest PoW block number is unknown")
	}
	// Only request deposits that have passed the ETH1 follow distance window.
	bNum = bNum.Sub(bNum, big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance)))
	allDeps := ps.beaconDB.AllDeposits(ctx, bNum)
	if len(allDeps) == 0 {
		return nil, nil
	}

	// Need to fetch if the deposits up to the state's latest eth 1 data matches
	// the number of all deposits in this RPC call. If not, then we return nil.
	beaconState, err := ps.beaconDB.HeadState(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not fetch beacon state: %v", err)
	}
	latestEth1DataHeight, err := ps.latestEth1Height(ctx, beaconState, currentVote)
	if err != nil {
		return nil, err
	}
	// If the state's latest eth1 data's block hash has a height of 100, we fetch all the deposits up to height 100.
	// If this doesn't match the number of deposits stored in the cache, the generated trie will not be the same and
	// root will fail to verify. This can happen in a scenario where we perhaps have a deposit from height 101,
	// so we want to avoid any possible mismatches in these lengths.
	upToEth1DataDeposits := ps.beaconDB.AllDeposits(ctx, latestEth1DataHeight)
	if len(upToEth1DataDeposits) != len(allDeps) {
		return nil, nil
	}
	depositData := [][]byte{}
	for _, dep := range upToEth1DataDeposits {
		depHash, err := ssz.HashTreeRoot(dep.Data)
		if err != nil {
			return nil, fmt.Errorf("coulf not hash deposit data %v", err)
		}
		depositData = append(depositData, depHash[:])
	}

	depositTrie, err := trieutil.GenerateTrieFromItems(depositData, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		return nil, fmt.Errorf("could not generate historical deposit trie from deposits: %v", err)
	}

	allPendingContainers := ps.beaconDB.PendingContainers(ctx, bNum)

	// Deposits need to be received in order of merkle index root, so this has to make sure
	// deposits are sorted from lowest to highest.
	var pendingDeps []*db.DepositContainer
	for _, dep := range allPendingContainers {
		if uint64(dep.Index) >= beaconState.Eth1DepositIndex {
			pendingDeps = append(pendingDeps, dep)
		}
	}

	for i := range pendingDeps {
		// Don't construct merkle proof if the number of deposits is more than max allowed in block.
		if uint64(i) == params.BeaconConfig().MaxDeposits {
			break
		}
		pendingDeps[i].Deposit, err = constructMerkleProof(depositTrie, pendingDeps[i].Index, pendingDeps[i].Deposit)
		if err != nil {
			return nil, err
		}
	}
	// Limit the return of pending deposits to not be more than max deposits allowed in block.
	var pendingDeposits []*ethpb.Deposit
	for i := 0; i < len(pendingDeps) && i < int(params.BeaconConfig().MaxDeposits); i++ {
		pendingDeposits = append(pendingDeposits, pendingDeps[i].Deposit)
	}
	return pendingDeposits, nil
}

// latestEth1Height determines what the latest eth1Blockhash is by tallying the votes in the
// beacon state
func (ps *ProposerServer) latestEth1Height(ctx context.Context, beaconState *pbp2p.BeaconState,
	currentVote *ethpb.Eth1Data) (*big.Int, error) {
	var eth1BlockHash [32]byte

	// Add in current vote, to get accurate vote tally
	beaconState.Eth1DataVotes = append(beaconState.Eth1DataVotes, currentVote)
	hasSupport, err := blocks.Eth1DataHasEnoughSupport(beaconState, currentVote)
	if err != nil {
		return nil, fmt.Errorf("could not determine if current eth1data vote has enough support: %v", err)
	}
	if hasSupport {
		eth1BlockHash = bytesutil.ToBytes32(currentVote.BlockHash)
	} else {
		eth1BlockHash = bytesutil.ToBytes32(beaconState.Eth1Data.BlockHash)
	}
	_, latestEth1DataHeight, err := ps.powChainService.BlockExists(ctx, eth1BlockHash)
	if err != nil {
		return nil, fmt.Errorf("could not fetch eth1data height: %v", err)
	}
	return latestEth1DataHeight, nil
}

// in case no vote for new eth1data vote considered best vote we
// default into returning the latest deposit root and the block
// hash of eth1 block hash that is FOLLOW_DISTANCE back from its
// latest block.
func (ps *ProposerServer) defaultEth1DataResponse(ctx context.Context, currentHeight *big.Int) (*ethpb.Eth1Data, error) {
	eth1FollowDistance := int64(params.BeaconConfig().Eth1FollowDistance)
	ancestorHeight := big.NewInt(0).Sub(currentHeight, big.NewInt(eth1FollowDistance))
	blockHash, err := ps.powChainService.BlockHashByHeight(ctx, ancestorHeight)
	if err != nil {
		return nil, fmt.Errorf("could not fetch ETH1_FOLLOW_DISTANCE ancestor: %v", err)
	}
	// Fetch all historical deposits up to an ancestor height.
	depositsTillHeight, depositRoot := ps.beaconDB.DepositsNumberAndRootAtHeight(ctx, ancestorHeight)
	if depositsTillHeight == 0 {
		return ps.powChainService.ChainStartETH1Data(), nil
	}
	return &ethpb.Eth1Data{
		DepositRoot:  depositRoot[:],
		BlockHash:    blockHash[:],
		DepositCount: depositsTillHeight,
	}, nil
}
