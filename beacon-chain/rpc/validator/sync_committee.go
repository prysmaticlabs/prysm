package validator

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GetSyncMessageBlockRoot retrieves the sync committee block root of the beacon chain.
func (vs *Server) GetSyncMessageBlockRoot(ctx context.Context, _ *emptypb.Empty) (*ethpb.SyncMessageBlockRootResponse, error) {
	r, err := vs.HeadFetcher.HeadRoot(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not retrieve head root: %v", err)
	}

	return &ethpb.SyncMessageBlockRootResponse{
		Root: r,
	}, nil
}

// SubmitSyncMessage submits the sync committee message to the network.
// It also saves the sync committee message into the pending pool for block inclusion.
func (vs *Server) SubmitSyncMessage(ctx context.Context, msg *ethpb.SyncCommitteeMessage) (*emptypb.Empty, error) {
	errs, ctx := errgroup.WithContext(ctx)

	// Broadcasting and saving message into the pool in parallel. As one fail should not affect another.
	errs.Go(func() error {
		return vs.P2P.Broadcast(ctx, msg)
	})

	if err := vs.SyncCommitteePool.SaveSyncCommitteeMessage(msg); err != nil {
		return nil, err
	}

	// Wait for p2p broadcast to complete and return the first error (if any)
	err := errs.Wait()
	return nil, err
}

// GetSyncSubcommitteeIndex is called by a sync committee participant to get its subcommittee index for sync message aggregation duty.
func (vs *Server) GetSyncSubcommitteeIndex(ctx context.Context, req *ethpb.SyncSubcommitteeIndexRequest) (*ethpb.SyncSubcommitteeIndexRespond, error) {
	indices, err := vs.syncSubcommitteeIndex(ctx, bytesutil.ToBytes48(req.PublicKey), req.Slot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get sync subcommittee index: %v", err)
	}
	return &ethpb.SyncSubcommitteeIndexRespond{
		Indices: indices,
	}, nil
}

// syncSubcommitteeIndex returns a list of subcommittee index of a validator and slot for sync message aggregation duty.
func (vs *Server) syncSubcommitteeIndex(ctx context.Context, pubkey [48]byte, slot types.Slot) ([]uint64, error) {
	headState, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, err
	}
	if slot > headState.Slot() {
		headState, err = state.ProcessSlots(ctx, headState, slot)
		if err != nil {
			return nil, err
		}
	}

	nextSlotEpoch := helpers.SlotToEpoch(headState.Slot() + 1)
	currentEpoch := helpers.CurrentEpoch(headState)

	switch {
	case altair.SyncCommitteePeriod(nextSlotEpoch) == altair.SyncCommitteePeriod(currentEpoch):
		committee, err := headState.CurrentSyncCommittee()
		if err != nil {
			return nil, err
		}
		indices, err := helpers.CurrentEpochSyncSubcommitteeIndices(committee, pubkey)
		if err != nil {
			return nil, err
		}
		return indices, nil
	// At sync committee period boundary, validator should sample the next epoch sync committee.
	case altair.SyncCommitteePeriod(nextSlotEpoch) == altair.SyncCommitteePeriod(currentEpoch)+1:
		committee, err := headState.NextSyncCommittee()
		if err != nil {
			return nil, err
		}
		indices, err := helpers.NextEpochSyncSubcommitteeIndices(committee, pubkey)
		if err != nil {
			return nil, err
		}
		return indices, nil
	default:
		// Impossible condition.
		return nil, errors.New("could get calculate sync subcommittee based on the period")
	}
}

// GetSyncCommitteeContribution is called by a sync committee aggregator to retrieve sync committee contribution object.
func (vs *Server) GetSyncCommitteeContribution(ctx context.Context, req *ethpb.SyncCommitteeContributionRequest) (*ethpb.SyncCommitteeContribution, error) {
	msgs, err := vs.SyncCommitteePool.SyncCommitteeMessages(req.Slot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get sync subcommittee messages: %v", err)
	}
	headRoot, err := vs.HeadFetcher.HeadRoot(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head root: %v", err)
	}
	headState, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}

	subCommitteeSize := params.BeaconConfig().SyncCommitteeSize / params.BeaconConfig().SyncCommitteeSubnetCount
	sigs := make([]bls.Signature, 0, subCommitteeSize)
	bits := bitfield.NewBitvector128()
	for _, msg := range msgs {
		if bytes.Equal(headRoot, msg.BlockRoot) {
			v, err := headState.ValidatorAtIndexReadOnly(msg.ValidatorIndex)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not get validator at index: %v", err)
			}
			indices, err := vs.syncSubcommitteeIndex(ctx, v.PublicKey(), req.Slot)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not get sync subcommittee index: %v", err)
			}
			for _, index := range indices {
				subnetIndex := index / subCommitteeSize
				if subnetIndex == req.SubnetId {
					bits.SetBitAt(index%subCommitteeSize, true)
					sig, err := bls.SignatureFromBytes(msg.Signature)
					if err != nil {
						return nil, status.Errorf(codes.Internal, "Could not get bls signature from bytes: %v", err)
					}
					sigs = append(sigs, sig)
				}
			}
		}
	}
	aggregatedSig := bls.AggregateSignatures(sigs)
	contribution := &ethpb.SyncCommitteeContribution{
		Slot:              headState.Slot(),
		BlockRoot:         headRoot,
		SubcommitteeIndex: req.SubnetId,
		AggregationBits:   bits,
		Signature:         aggregatedSig.Marshal(),
	}

	return contribution, nil
}

// SubmitSignedContributionAndProof is called by a sync committee aggregator to submit signed contribution and proof object.
func (vs *Server) SubmitSignedContributionAndProof(ctx context.Context, s *ethpb.SignedContributionAndProof) (*emptypb.Empty, error) {
	errs, ctx := errgroup.WithContext(ctx)

	// Broadcasting and saving contribution into the pool in parallel. As one fail should not affect another.
	errs.Go(func() error {
		return vs.P2P.Broadcast(ctx, s)
	})

	if err := vs.SyncCommitteePool.SaveSyncCommitteeContribution(s.Message.Contribution); err != nil {
		return nil, err
	}

	// Wait for p2p broadcast to complete and return the first error (if any)
	err := errs.Wait()
	return nil, err
}
