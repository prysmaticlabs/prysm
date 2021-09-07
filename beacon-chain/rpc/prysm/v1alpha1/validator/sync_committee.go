package validator

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GetSyncMessageBlockRoot retrieves the sync committee block root of the beacon chain.
func (vs *Server) GetSyncMessageBlockRoot(
	ctx context.Context, _ *emptypb.Empty,
) (*ethpb.SyncMessageBlockRootResponse, error) {
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

	idxResp, err := vs.syncSubcommitteeIndex(ctx, msg.ValidatorIndex, msg.Slot)
	if err != nil {
		return &emptypb.Empty{}, err
	}
	// Broadcasting and saving message into the pool in parallel. As one fail should not affect another.
	// This broadcasts for all subnets.
	for _, id := range idxResp.Indices {
		subCommitteeSize := params.BeaconConfig().SyncCommitteeSize / params.BeaconConfig().SyncCommitteeSubnetCount
		subnet := uint64(id) / subCommitteeSize
		errs.Go(func() error {
			return vs.P2P.BroadcastSyncCommitteeMessage(ctx, subnet, msg)
		})
	}

	if err := vs.SyncCommitteePool.SaveSyncCommitteeMessage(msg); err != nil {
		return &emptypb.Empty{}, err
	}

	// Wait for p2p broadcast to complete and return the first error (if any)
	err = errs.Wait()
	return &emptypb.Empty{}, err
}

// GetSyncSubcommitteeIndex is called by a sync committee participant to get
// its subcommittee index for sync message aggregation duty.
func (vs *Server) GetSyncSubcommitteeIndex(
	ctx context.Context, req *ethpb.SyncSubcommitteeIndexRequest,
) (*ethpb.SyncSubcommitteeIndexResponse, error) {
	index, exists := vs.HeadFetcher.HeadPublicKeyToValidatorIndex(ctx, bytesutil.ToBytes48(req.PublicKey))
	if !exists {
		return nil, errors.New("public key does not exist in state")
	}
	indices, err := vs.syncSubcommitteeIndex(ctx, index, req.Slot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get sync subcommittee index: %v", err)
	}
	return indices, nil
}

// syncSubcommitteeIndex returns a list of subcommittee index of a validator and slot for sync message aggregation duty.
func (vs *Server) syncSubcommitteeIndex(
	ctx context.Context, index types.ValidatorIndex, slot types.Slot,
) (*ethpb.SyncSubcommitteeIndexResponse, error) {

	nextSlotEpoch := core.SlotToEpoch(slot + 1)
	currentEpoch := core.SlotToEpoch(slot)

	switch {
	case core.SyncCommitteePeriod(nextSlotEpoch) == core.SyncCommitteePeriod(currentEpoch):
		indices, err := vs.HeadFetcher.HeadCurrentSyncCommitteeIndices(ctx, index, slot)
		if err != nil {
			return nil, err
		}
		return &ethpb.SyncSubcommitteeIndexResponse{
			Indices: indices,
		}, nil
	// At sync committee period boundary, validator should sample the next epoch sync committee.
	case core.SyncCommitteePeriod(nextSlotEpoch) == core.SyncCommitteePeriod(currentEpoch)+1:
		indices, err := vs.HeadFetcher.HeadNextSyncCommitteeIndices(ctx, index, slot)
		if err != nil {
			return nil, err
		}
		return &ethpb.SyncSubcommitteeIndexResponse{
			Indices: indices,
		}, nil
	default:
		// Impossible condition.
		return nil, errors.New("could get calculate sync subcommittee based on the period")
	}
}

// GetSyncCommitteeContribution is called by a sync committee aggregator
// to retrieve sync committee contribution object.
func (vs *Server) GetSyncCommitteeContribution(
	ctx context.Context, req *ethpb.SyncCommitteeContributionRequest,
) (*ethpb.SyncCommitteeContribution, error) {
	msgs, err := vs.SyncCommitteePool.SyncCommitteeMessages(req.Slot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get sync subcommittee messages: %v", err)
	}
	headRoot, err := vs.HeadFetcher.HeadRoot(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head root: %v", err)
	}
	aggregatedSig, bits, err := vs.AggregatedSigAndAggregationBits(ctx, msgs, req.Slot, req.SubnetId, headRoot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get contribution data: %v", err)
	}
	contribution := &ethpb.SyncCommitteeContribution{
		Slot:              req.Slot,
		BlockRoot:         headRoot,
		SubcommitteeIndex: req.SubnetId,
		AggregationBits:   bits,
		Signature:         aggregatedSig,
	}

	return contribution, nil
}

// SubmitSignedContributionAndProof is called by a sync committee aggregator
// to submit signed contribution and proof object.
func (vs *Server) SubmitSignedContributionAndProof(
	ctx context.Context, s *ethpb.SignedContributionAndProof,
) (*emptypb.Empty, error) {
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
	return &emptypb.Empty{}, err
}

// AggregatedSigAndAggregationBits returns the aggregated signature and aggregation bits
// associated with a particular set of sync committee messages.
func (vs *Server) AggregatedSigAndAggregationBits(
	ctx context.Context,
	msgs []*ethpb.SyncCommitteeMessage,
	slot types.Slot,
	subnetId uint64,
	blockRoot []byte,
) ([]byte, []byte, error) {
	subCommitteeSize := params.BeaconConfig().SyncCommitteeSize / params.BeaconConfig().SyncCommitteeSubnetCount
	sigs := make([]bls.Signature, 0, subCommitteeSize)
	bits := ethpb.NewSyncCommitteeAggregationBits()
	for _, msg := range msgs {
		if bytes.Equal(blockRoot, msg.BlockRoot) {
			idxResp, err := vs.syncSubcommitteeIndex(ctx, msg.ValidatorIndex, slot)
			if err != nil {
				return []byte{}, nil, errors.Wrapf(err, "could not get sync subcommittee index")
			}
			for _, index := range idxResp.Indices {
				i := uint64(index)
				subnetIndex := i / subCommitteeSize
				if subnetIndex == subnetId {
					bits.SetBitAt(i%subCommitteeSize, true)
					sig, err := bls.SignatureFromBytes(msg.Signature)
					if err != nil {
						return []byte{}, nil, errors.Wrapf(err, "Could not get bls signature from bytes")
					}
					sigs = append(sigs, sig)
				}
			}
		}
	}
	aggregatedSig := make([]byte, 96)
	aggregatedSig[0] = 0xC0
	if len(sigs) != 0 {
		aggregatedSig = bls.AggregateSignatures(sigs).Marshal()
	}

	return aggregatedSig, bits, nil
}
