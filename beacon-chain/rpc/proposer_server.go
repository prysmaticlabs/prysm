package rpc

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
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

// ProposerIndex sends a response to the client which returns the proposer index for a given slot. Validators
// are shuffled and assigned slots to attest/propose to. This method will look for the validator that is assigned
// to propose a beacon block at the given slot.
func (ps *ProposerServer) ProposerIndex(ctx context.Context, req *pb.ProposerIndexRequest) (*pb.ProposerIndexResponse, error) {
	beaconState, err := ps.beaconDB.State(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get beacon state: %v", err)
	}

	proposerIndex, err := helpers.BeaconProposerIndex(
		beaconState,
		req.SlotNumber,
	)
	if err != nil {
		return nil, fmt.Errorf("could not get index of previous proposer: %v", err)
	}

	return &pb.ProposerIndexResponse{
		Index: proposerIndex,
	}, nil
}

// ProposeBlock is called by a proposer during its assigned slot to create a block in an attempt
// to get it processed by the beacon node as the canonical head.
func (ps *ProposerServer) ProposeBlock(ctx context.Context, blk *pbp2p.BeaconBlock) (*pb.ProposeResponse, error) {
	h, err := hashutil.HashBeaconBlock(blk)
	if err != nil {
		return nil, fmt.Errorf("could not tree hash block: %v", err)
	}
	log.WithField("blockRoot", fmt.Sprintf("%#x", h)).Debugf("Block proposal received via RPC")
	beaconState, err := ps.chainService.ReceiveBlock(ctx, blk)
	if err != nil {
		return nil, fmt.Errorf("could not process beacon block: %v", err)
	}
	if err := ps.chainService.ApplyForkChoiceRule(ctx, blk, beaconState); err != nil {
		return nil, fmt.Errorf("could not apply fork choice rule: %v", err)
	}
	return &pb.ProposeResponse{BlockRootHash32: h[:]}, nil
}

// PendingAttestations retrieves attestations kept in the beacon node's operations pool which have
// not yet been included into the beacon chain. Proposers include these pending attestations in their
// proposed blocks when performing their responsibility. If desired, callers can choose to filter pending
// attestations which are ready for inclusion. That is, attestations that satisfy:
// attestation.slot + MIN_ATTESTATION_INCLUSION_DELAY <= state.slot.
func (ps *ProposerServer) PendingAttestations(ctx context.Context, req *pb.PendingAttestationsRequest) (*pb.PendingAttestationsResponse, error) {
	beaconState, err := ps.beaconDB.State(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve beacon state: %v", err)
	}
	atts, err := ps.operationService.PendingAttestations()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve pending attestations from operations service: %v", err)
	}

	head, err := ps.beaconDB.ChainHead()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve chain head: %v", err)
	}
	blockRoot, err := hashutil.HashBeaconBlock(head)
	if err != nil {
		return nil, fmt.Errorf("could not hash beacon block: %v", err)
	}
	for beaconState.Slot < req.ProposalBlockSlot {
		beaconState, err = state.ExecuteStateTransition(
			ctx, beaconState, nil /* block */, blockRoot, &state.TransitionConfig{},
		)
		if err != nil {
			return nil, fmt.Errorf("could not execute head transition: %v", err)
		}
	}

	// Use the optional proposal block slot parameter as the current slot for
	// determining the validity window for attestations.
	currentSlot := req.ProposalBlockSlot
	if currentSlot == 0 {
		currentSlot = beaconState.Slot
	}

	// Remove any attestation from the list if their slot is before the start of
	// the previous epoch or does not match the current state previous justified
	// epoch. This should be handled in the operationService cleanup but we
	// should filter here in case it wasn't yet processed.
	boundary := currentSlot - params.BeaconConfig().SlotsPerEpoch
	attsWithinBoundary := make([]*pbp2p.Attestation, 0, len(atts))
	for _, att := range atts {

		var expectedJustifedEpoch uint64
		if helpers.SlotToEpoch(att.Data.Slot+1) >= helpers.SlotToEpoch(currentSlot) {
			expectedJustifedEpoch = beaconState.JustifiedEpoch
		} else {
			expectedJustifedEpoch = beaconState.PreviousJustifiedEpoch
		}

		if att.Data.Slot > boundary && att.Data.JustifiedEpoch == expectedJustifedEpoch {
			attsWithinBoundary = append(attsWithinBoundary, att)
		}
	}
	atts = attsWithinBoundary

	if req.FilterReadyForInclusion {
		var attsReadyForInclusion []*pbp2p.Attestation
		for _, val := range atts {
			if val.Data.Slot+params.BeaconConfig().MinAttestationInclusionDelay <= currentSlot {
				attsReadyForInclusion = append(attsReadyForInclusion, val)
			}
		}
		return &pb.PendingAttestationsResponse{
			PendingAttestations: attsReadyForInclusion,
		}, nil
	}
	return &pb.PendingAttestationsResponse{
		PendingAttestations: atts,
	}, nil
}

// ComputeStateRoot computes the state root after a block has been processed through a state transition and
// returns it to the validator client.
func (ps *ProposerServer) ComputeStateRoot(ctx context.Context, req *pbp2p.BeaconBlock) (*pb.StateRootResponse, error) {
	beaconState, err := ps.beaconDB.State(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get beacon state: %v", err)
	}

	parentHash := bytesutil.ToBytes32(req.ParentRootHash32)
	// Check for skipped slots.
	for beaconState.Slot < req.Slot-1 {
		beaconState, err = state.ExecuteStateTransition(
			ctx,
			beaconState,
			nil,
			parentHash,
			state.DefaultConfig(),
		)
		if err != nil {
			return nil, fmt.Errorf("could not execute state transition %v", err)
		}
	}
	beaconState, err = state.ExecuteStateTransition(
		ctx,
		beaconState,
		req,
		parentHash,
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
