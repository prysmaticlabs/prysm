package rpc

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	p2p "github.com/prysmaticlabs/prysm/shared/deprecated-p2p"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// AttesterServer defines a server implementation of the gRPC Attester service,
// providing RPC methods for validators acting as attesters to broadcast votes on beacon blocks.
type AttesterServer struct {
	p2p              p2p.Broadcaster
	beaconDB         *db.BeaconDB
	operationService operationService
	cache            *cache.AttestationCache
}

// SubmitAttestation is a function called by an attester in a sharding validator to vote
// on a block via an attestation object as defined in the Ethereum Serenity specification.
func (as *AttesterServer) SubmitAttestation(ctx context.Context, att *ethpb.Attestation) (*pb.AttestResponse, error) {
	if err := as.operationService.HandleAttestation(ctx, att); err != nil {
		return nil, err
	}

	// Update attestation target for RPC server to run necessary fork choice.
	// We need to retrieve the head block to get its parent root.
	head, err := as.beaconDB.Block(bytesutil.ToBytes32(att.Data.BeaconBlockRoot))
	if err != nil {
		return nil, err
	}
	// If the head block is nil, we can't save the attestation target.
	if head == nil {
		return nil, fmt.Errorf("could not find head %#x in db", bytesutil.Trunc(att.Data.BeaconBlockRoot))
	}
	// TODO(#3088): Remove this when fork-choice is updated to the new one.
	attestationSlot := att.Data.Target.Epoch * params.BeaconConfig().SlotsPerEpoch
	attTarget := &pbp2p.AttestationTarget{
		Slot:            attestationSlot,
		BeaconBlockRoot: att.Data.BeaconBlockRoot,
		ParentRoot:      head.ParentRoot,
	}
	if err := as.beaconDB.SaveAttestationTarget(ctx, attTarget); err != nil {
		return nil, fmt.Errorf("could not save attestation target")
	}

	as.p2p.Broadcast(ctx, att)

	hash, err := hashutil.HashProto(att)
	if err != nil {
		return nil, err
	}

	return &pb.AttestResponse{Root: hash[:]}, nil
}

// RequestAttestation requests that the beacon node produce an IndexedAttestation,
// with a blank signature field, which the validator will then sign.
func (as *AttesterServer) RequestAttestation(ctx context.Context, req *pb.AttestationRequest) (*ethpb.AttestationData, error) {
	res, err := as.cache.Get(ctx, req)
	if err != nil {
		return nil, err
	}

	if res != nil {
		return res, nil
	}

	if err := as.cache.MarkInProgress(req); err != nil {
		if err == cache.ErrAlreadyInProgress {
			res, err := as.cache.Get(ctx, req)
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
		if err := as.cache.MarkNotInProgress(req); err != nil {
			log.WithError(err).Error("Failed to mark cache not in progress")
		}
	}()

	// Set the attestation data's beacon block root = hash_tree_root(head) where head
	// is the validator's view of the head block of the beacon chain during the slot.
	headBlock, err := as.beaconDB.ChainHead()
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve chain head")
	}
	headRoot, err := ssz.SigningRoot(headBlock)
	if err != nil {
		return nil, errors.Wrap(err, "could not tree hash beacon block")
	}

	// Let head state be the state of head block processed through empty slots up to assigned slot.
	headState, err := as.beaconDB.HeadState(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not fetch head state")
	}

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
		return nil, fmt.Errorf("could not tree hash crosslink for shard %d: %v",
			req.Shard, err)
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

	if err := as.cache.Put(ctx, req, res); err != nil {
		return nil, err
	}

	return res, nil
}
