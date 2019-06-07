package rpc

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
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

// ProposeBlock is called by a proposer during its assigned slot to create a block in an attempt
// to get it processed by the beacon node as the canonical head.
func (ps *ProposerServer) ProposeBlock(ctx context.Context, blk *pbp2p.BeaconBlock) (*pb.ProposeResponse, error) {
	h, err := hashutil.HashBeaconBlock(blk)
	if err != nil {
		return nil, fmt.Errorf("could not tree hash block: %v", err)
	}
	log.WithField("blockRoot", fmt.Sprintf("%#x", bytesutil.Trunc(h[:]))).Debugf(
		"Block proposal received via RPC")
	beaconState, err := ps.chainService.ReceiveBlock(ctx, blk)
	if err != nil {
		return nil, fmt.Errorf("could not process beacon block: %v", err)
	}
	if err := ps.beaconDB.UpdateChainHead(ctx, blk, beaconState); err != nil {
		return nil, fmt.Errorf("failed to update chain: %v", err)
	}
	ps.chainService.UpdateCanonicalRoots(blk, h)
	log.WithFields(logrus.Fields{
		"headRoot": fmt.Sprintf("%#x", bytesutil.Trunc(h[:])),
		"headSlot": blk.Slot,
	}).Info("Chain head block and state updated")
	return &pb.ProposeResponse{BlockRootHash32: h[:]}, nil
}

// PendingAttestations retrieves attestations kept in the beacon node's operations pool which have
// not yet been included into the beacon chain. Proposers include these pending attestations in their
// proposed blocks when performing their responsibility. If desired, callers can choose to filter pending
// attestations which are ready for inclusion. That is, attestations that satisfy:
// attestation.slot + MIN_ATTESTATION_INCLUSION_DELAY <= state.slot.
func (ps *ProposerServer) PendingAttestations(ctx context.Context, req *pb.PendingAttestationsRequest) (*pb.PendingAttestationsResponse, error) {
	beaconState, err := ps.beaconDB.HeadState(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve beacon state: %v", err)
	}
	atts, err := ps.operationService.PendingAttestations(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve pending attestations from operations service: %v", err)
	}
	beaconState.Slot++

	var attsReadyForInclusion []*pbp2p.Attestation
	for _, att := range atts {
		slot, err := helpers.AttestationDataSlot(beaconState, att.Data)
		if err != nil {
			return nil, fmt.Errorf("could not get attestation slot: %v", err)
		}
		if slot+params.BeaconConfig().MinAttestationInclusionDelay <= beaconState.Slot {
			attsReadyForInclusion = append(attsReadyForInclusion, att)
		}
	}

	validAtts := make([]*pbp2p.Attestation, 0, len(attsReadyForInclusion))
	for _, att := range attsReadyForInclusion {
		slot, err := helpers.AttestationDataSlot(beaconState, att.Data)
		if err != nil {
			return nil, fmt.Errorf("could not get attestation slot: %v", err)
		}

		if _, err := blocks.ProcessAttestation(beaconState, att, false); err != nil {
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

	return &pb.PendingAttestationsResponse{
		PendingAttestations: validAtts,
	}, nil
}

// Eth1Data is a mechanism used by block proposers vote on a recent Ethereum 1.0 block hash and an
// associated deposit root found in the Ethereum 1.0 deposit contract. When consensus is formed,
// state.latest_eth1_data is updated, and validator deposits up to this root can be processed.
// The deposit root can be calculated by calling the get_deposit_root() function of
// the deposit contract using the post-state of the block hash.
//
// TODO(#2307): Refactor for v0.6.
func (ps *ProposerServer) Eth1Data(ctx context.Context, _ *ptypes.Empty) (*pb.Eth1DataResponse, error) {
	return nil, nil
}

// ComputeStateRoot computes the state root after a block has been processed through a state transition and
// returns it to the validator client.
func (ps *ProposerServer) ComputeStateRoot(ctx context.Context, req *pbp2p.BeaconBlock) (*pb.StateRootResponse, error) {
	if !featureconfig.FeatureConfig().EnableComputeStateRoot {
		log.Debug("Compute state root disabled, returning no-op result")
		return &pb.StateRootResponse{StateRoot: []byte("no-op")}, nil
	}

	beaconState, err := ps.beaconDB.HeadState(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get beacon state: %v", err)
	}

	beaconState, err = state.ExecuteStateTransition(
		ctx,
		beaconState,
		req,
		state.DefaultConfig(),
	)
	if err != nil {
		return nil, fmt.Errorf("could not execute state transition %v", err)
	}

	beaconStateHash, err := hashutil.HashProto(beaconState)
	if err != nil {
		return nil, fmt.Errorf("could not tree hash beacon state: %v", err)
	}
	log.WithField("beaconStateHash", fmt.Sprintf("%#x", beaconStateHash)).Debugf("Computed state hash")
	return &pb.StateRootResponse{
		StateRoot: beaconStateHash[:],
	}, nil
}

// PendingDeposits returns a list of pending deposits that are ready for
// inclusion in the next beacon block.
func (ps *ProposerServer) PendingDeposits(ctx context.Context, _ *ptypes.Empty) (*pb.PendingDepositsResponse, error) {
	bNum := ps.powChainService.LatestBlockHeight()
	if bNum == nil {
		return nil, errors.New("latest PoW block number is unknown")
	}
	// Only request deposits that have passed the ETH1 follow distance window.
	bNum = bNum.Sub(bNum, big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance)))
	allDeps := ps.beaconDB.AllDeposits(ctx, bNum)
	if len(allDeps) == 0 {
		return &pb.PendingDepositsResponse{PendingDeposits: nil}, nil
	}

	// Need to fetch if the deposits up to the state's latest eth 1 data matches
	// the number of all deposits in this RPC call. If not, then we return nil.
	beaconState, err := ps.beaconDB.HeadState(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not fetch beacon state: %v", err)
	}
	h := bytesutil.ToBytes32(beaconState.LatestEth1Data.BlockRoot)
	_, latestEth1DataHeight, err := ps.powChainService.BlockExists(ctx, h)
	if err != nil {
		return nil, fmt.Errorf("could not fetch eth1data height: %v", err)
	}
	// If the state's latest eth1 data's block hash has a height of 100, we fetch all the deposits up to height 100.
	// If this doesn't match the number of deposits stored in the cache, the generated trie will not be the same and
	// root will fail to verify. This can happen in a scenario where we perhaps have a deposit from height 101,
	// so we want to avoid any possible mismatches in these lengths.
	upToLatestEth1DataDeposits := ps.beaconDB.AllDeposits(ctx, latestEth1DataHeight)
	if len(upToLatestEth1DataDeposits) != len(allDeps) {
		return &pb.PendingDepositsResponse{PendingDeposits: nil}, nil
	}
	depositData := [][]byte{}
	for _, dep := range upToLatestEth1DataDeposits {
		depHash, err := hashutil.DepositHash(dep.Data)
		if err != nil {
			return nil, fmt.Errorf("coulf not hash deposit data %v", err)
		}
		depositData = append(depositData, depHash[:])
	}

	depositTrie, err := trieutil.GenerateTrieFromItems(depositData, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		return nil, fmt.Errorf("could not generate historical deposit trie from deposits: %v", err)
	}

	allPendingDeps := ps.beaconDB.PendingDeposits(ctx, bNum)

	// Deposits need to be received in order of merkle index root, so this has to make sure
	// deposits are sorted from lowest to highest.
	var pendingDeps []*pbp2p.Deposit
	for _, dep := range allPendingDeps {
		if dep.Index >= beaconState.DepositIndex {
			pendingDeps = append(pendingDeps, dep)
		}
	}

	for i := range pendingDeps {
		// Don't construct merkle proof if the number of deposits is more than max allowed in block.
		if uint64(i) == params.BeaconConfig().MaxDeposits {
			break
		}
		pendingDeps[i], err = constructMerkleProof(depositTrie, pendingDeps[i])
		if err != nil {
			return nil, err
		}
	}
	// Limit the return of pending deposits to not be more than max deposits allowed in block.
	var pendingDeposits []*pbp2p.Deposit
	for i := 0; i < len(pendingDeps) && i < int(params.BeaconConfig().MaxDeposits); i++ {
		pendingDeposits = append(pendingDeposits, pendingDeps[i])
	}
	return &pb.PendingDepositsResponse{PendingDeposits: pendingDeposits}, nil
}
