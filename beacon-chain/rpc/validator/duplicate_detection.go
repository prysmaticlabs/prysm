package validator

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/interop"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/interfaces"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Number of Epochs to check
var NoEpochsToChech uint64

// ADD COMMENTS
func (vs *Server) DetectDoppelganger(ctx context.Context, req *ethpb.DetectDoppelgangerRequest)(*ethpb.DetectDoppelgangerResponse, error) {
	log.Info("Doppelganger rpc service started")

	NoEpochsToChech = params.BeaconConfig().DuplicateValidatorEpochsCheck


	// see reward_penalty.go precompute
	baseRewardFactor := params.BeaconConfig().BaseRewardFactor
	baseRewardsPerEpoch := params.BeaconConfig().BaseRewardsPerEpoch
	proposerRewardQuotient := params.BeaconConfig().ProposerRewardQuotient

	ctx, span := trace.StartSpan(ctx, "ProposerServer.GetBlock")
	defer span.End()
	span.AddAttributes(trace.Int64Attribute("slot", int64(req.Slot)))

	if vs.SyncChecker.Syncing() {
		return nil, status.Errorf(codes.Unavailable, "Syncing to latest head, not ready to respond")
	}

	// Retrieve the parent block as the current head of the canonical chain.
	parentRoot, err := vs.HeadFetcher.HeadRoot(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not retrieve head root: %v", err)
	}

	head, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state %v", err)
	}

	if featureconfig.Get().EnableNextSlotStateCache {
		head, err = state.ProcessSlotsUsingNextSlotCache(ctx, head, parentRoot, req.Slot)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not advance slots to calculate proposer index: %v", err)
		}
	} else {
		head, err = state.ProcessSlots(ctx, head, req.Slot)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not advance slot to calculate proposer index: %v", err)
		}
	}

	eth1Data, err := vs.eth1DataMajorityVote(ctx, head)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get ETH1 data: %v", err)
	}

	// Pack ETH1 deposits which have not been included in the beacon chain.
	deposits, err := vs.deposits(ctx, head, eth1Data)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get ETH1 deposits: %v", err)
	}

	// Pack aggregated attestations which have not been included in the beacon chain.
	atts, err := vs.packAttestations(ctx, head)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get attestations to pack into block: %v", err)
	}

	// Use zero hash as stub for state root to compute later.
	stateRoot := params.BeaconConfig().ZeroHash[:]

	graffiti := bytesutil.ToBytes32(req.Graffiti)

	// Calculate new proposer index.
	idx, err := helpers.BeaconProposerIndex(head)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not calculate proposer index %v", err)
	}

	blk := &ethpb.BeaconBlock{
		Slot:          req.Slot,
		ParentRoot:    parentRoot,
		StateRoot:     stateRoot,
		ProposerIndex: idx,
		Body: &ethpb.BeaconBlockBody{
			Eth1Data:          eth1Data,
			Deposits:          deposits,
			Attestations:      atts,
			RandaoReveal:      req.RandaoReveal,
			ProposerSlashings: vs.SlashingsPool.PendingProposerSlashings(ctx, head, false /*noLimit*/),
			AttesterSlashings: vs.SlashingsPool.PendingAttesterSlashings(ctx, head, false /*noLimit*/),
			VoluntaryExits:    vs.ExitPool.PendingExits(head, req.Slot, false /*noLimit*/),
			Graffiti:          graffiti[:],
		},
	}

	// Compute state root with the newly constructed block.
	stateRoot, err = vs.computeStateRoot(ctx, interfaces.WrappedPhase0SignedBeaconBlock(&ethpb.SignedBeaconBlock{Block: blk, Signature: make([]byte, 96)}))
	if err != nil {
		interop.WriteBlockToDisk(interfaces.WrappedPhase0SignedBeaconBlock(&ethpb.SignedBeaconBlock{Block: blk}), true /*failed*/)
		return nil, status.Errorf(codes.Internal, "Could not compute state root: %v", err)
	}
	blk.StateRoot = stateRoot

	return nil, nil
}

