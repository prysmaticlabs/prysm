package rpc

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

// AttesterServer defines a server implementation of the gRPC Attester service,
// providing RPC methods for validators acting as attesters to broadcast votes on beacon blocks.
type AttesterServer struct {
	p2p               p2p.Broadcaster
	beaconDB          db.Database
	operationsHandler operations.Handler
	attReceiver       blockchain.AttestationReceiver
	headFetcher       blockchain.HeadFetcher
	attestationCache  *cache.AttestationCache
}

// SubmitAttestation is a function called by an attester in a sharding validator to vote
// on a block via an attestation object as defined in the Ethereum Serenity specification.
func (as *AttesterServer) SubmitAttestation(ctx context.Context, att *ethpb.Attestation) (*pb.AttestResponse, error) {
	root, err := ssz.SigningRoot(att)
	if err != nil {
		return nil, errors.Wrap(err, "failed to sign root attestation")
	}

	go func() {
		ctx = trace.NewContext(context.Background(), trace.FromContext(ctx))
		if err := as.operationsHandler.HandleAttestation(ctx, att); err != nil {
			log.WithError(err).Error("could not handle attestation in operations service")
			return
		}
		if err := as.attReceiver.ReceiveAttestation(ctx, att); err != nil {
			log.WithError(err).Error("could not receive attestation in chain service")
		}
	}()

	return &pb.AttestResponse{Root: root[:]}, nil
}

// RequestAttestation requests that the beacon node produce an IndexedAttestation,
// with a blank signature field, which the validator will then sign.
func (as *AttesterServer) RequestAttestation(ctx context.Context, req *pb.AttestationRequest) (*ethpb.AttestationData, error) {
	ctx, span := trace.StartSpan(ctx, "AttesterServer.RequestAttestation")
	defer span.End()
	span.AddAttributes(
		trace.Int64Attribute("slot", int64(req.Slot)),
		trace.Int64Attribute("shard", int64(req.Shard)),
	)
	res, err := as.attestationCache.Get(ctx, req)
	if err != nil {
		return nil, err
	}

	if res != nil {
		return res, nil
	}

	if err := as.attestationCache.MarkInProgress(req); err != nil {
		if err == cache.ErrAlreadyInProgress {
			res, err := as.attestationCache.Get(ctx, req)
			if err != nil {
				return nil, err
			}

			if res == nil {
				return nil, errors.New("a request was in progress and resolved to nil")
			}
			return res, nil
		}
		return nil, err
	}
	defer func() {
		if err := as.attestationCache.MarkNotInProgress(req); err != nil {
			log.WithError(err).Error("Failed to mark cache not in progress")
		}
	}()

	headState := as.headFetcher.HeadState()
	headRoot := as.headFetcher.HeadRoot()

	headState, err = state.ProcessSlots(ctx, headState, req.Slot)
	if err != nil {
		return nil, errors.Wrapf(err, "could not process slots up to %d", req.Slot)
	}

	targetEpoch := helpers.CurrentEpoch(headState)
	epochStartSlot := helpers.StartSlot(targetEpoch)
	targetRoot := make([]byte, 32)
	if epochStartSlot == headState.Slot {
		targetRoot = headRoot[:]
	} else {
		targetRoot, err = helpers.BlockRootAtSlot(headState, epochStartSlot)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get target block for slot %d", epochStartSlot)
		}
	}

	startEpoch := headState.CurrentCrosslinks[req.Shard].EndEpoch
	endEpoch := startEpoch + params.BeaconConfig().MaxEpochsPerCrosslink
	if endEpoch > targetEpoch {
		endEpoch = targetEpoch
	}
	crosslinkRoot, err := ssz.HashTreeRoot(headState.CurrentCrosslinks[req.Shard])
	if err != nil {
		return nil, errors.Wrapf(err, "could not tree hash crosslink for shard %d", req.Shard)
	}
	res = &ethpb.AttestationData{
		BeaconBlockRoot: headRoot[:],
		Source:          headState.CurrentJustifiedCheckpoint,
		Target: &ethpb.Checkpoint{
			Epoch: targetEpoch,
			Root:  targetRoot,
		},
		Crosslink: &ethpb.Crosslink{
			Shard:      req.Shard,
			StartEpoch: startEpoch,
			EndEpoch:   endEpoch,
			ParentRoot: crosslinkRoot[:],
			DataRoot:   params.BeaconConfig().ZeroHash[:],
		},
	}

	if err := as.attestationCache.Put(ctx, req, res); err != nil {
		return nil, err
	}

	return res, nil
}
