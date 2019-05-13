package rpc

import (
	"context"
	"errors"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/p2p"
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

// AttestHead is a function called by an attester in a sharding validator to vote
// on a block via an attestation object as defined in the Ethereum Serenity specification.
func (as *AttesterServer) AttestHead(ctx context.Context, att *pbp2p.Attestation) (*pb.AttestResponse, error) {
	h, err := hashutil.HashProto(att)
	if err != nil {
		return nil, fmt.Errorf("could not hash attestation: %v", err)
	}

	if err := as.operationService.HandleAttestations(ctx, att); err != nil {
		return nil, err
	}

	// Update attestation target for RPC server to run necessary fork choice.
	// We need to retrieve the head block to get its parent root.
	head, err := as.beaconDB.Block(bytesutil.ToBytes32(att.Data.BeaconBlockRootHash32))
	if err != nil {
		return nil, err
	}
	// If the head block is nil, we can't save the attestation target.
	if head == nil {
		return nil, fmt.Errorf("could not find head %#x in db", bytesutil.Trunc(att.Data.BeaconBlockRootHash32))
	}
	attTarget := &pbp2p.AttestationTarget{
		Slot:       att.Data.Slot,
		BlockRoot:  att.Data.BeaconBlockRootHash32,
		ParentRoot: head.ParentRootHash32,
	}
	if err := as.beaconDB.SaveAttestationTarget(ctx, attTarget); err != nil {
		return nil, fmt.Errorf("could not save attestation target")
	}

	as.p2p.Broadcast(ctx, &pbp2p.AttestationAnnounce{
		Hash: h[:],
	})
	return &pb.AttestResponse{AttestationHash: h[:]}, nil
}

// AttestationDataAtSlot fetches the necessary information from the current canonical head
// and beacon state for an assigned attester to perform necessary responsibilities. This includes
// fetching the epoch boundary roots, the latest justified block root, among others.
func (as *AttesterServer) AttestationDataAtSlot(ctx context.Context, req *pb.AttestationDataRequest) (*pb.AttestationDataResponse, error) {
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
	head, err := as.beaconDB.ChainHead()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve chain head: %v", err)
	}
	headRoot, err := hashutil.HashBeaconBlock(head)
	if err != nil {
		return nil, fmt.Errorf("could not tree hash beacon block: %v", err)
	}

	// Let head state be the state of head block processed through empty slots up to assigned slot.
	headState, err := as.beaconDB.HeadState(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not fetch head state: %v", err)
	}

	for headState.Slot < req.Slot {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		headState, err = state.ExecuteStateTransition(
			ctx, headState, nil /* block */, headRoot, state.DefaultConfig(),
		)
		if err != nil {
			return nil, fmt.Errorf("could not execute head transition: %v", err)
		}
	}

	// Fetch the epoch boundary root = hash_tree_root(epoch_boundary)
	// where epoch_boundary is the block at the most recent epoch boundary in the
	// chain defined by head -- i.e. the BeaconBlock where block.slot == get_epoch_start_slot(head.slot).
	// If the epoch boundary slot is the same as state current slot,
	// we set epoch boundary root to an empty root.
	epochBoundaryRoot := make([]byte, 32)
	epochStartSlot := helpers.StartSlot(helpers.SlotToEpoch(headState.Slot))
	if epochStartSlot == headState.Slot {
		epochBoundaryRoot = headRoot[:]
	} else {
		epochBoundaryRoot, err = blocks.BlockRoot(headState, epochStartSlot)
		if err != nil {
			return nil, fmt.Errorf("could not get epoch boundary block for slot %d: %v",
				epochStartSlot, err)
		}
	}
	// epoch_start_slot = get_epoch_start_slot(slot_to_epoch(head.slot))
	// Fetch the justified block root = hash_tree_root(justified_block) where
	// justified_block is the block at state.justified_epoch in the chain defined by head.
	// On the server side, this is fetched by calling get_block_root(state, justified_epoch).
	// If the last justified boundary slot is the same as state current slot (ex: slot 0),
	// we set justified block root to an empty root.
	justifiedBlockRoot := headState.JustifiedRoot

	// If an attester has to attest for genesis block.
	if headState.Slot == params.BeaconConfig().GenesisSlot {
		epochBoundaryRoot = params.BeaconConfig().ZeroHash[:]
		justifiedBlockRoot = params.BeaconConfig().ZeroHash[:]
	}

	res = &pb.AttestationDataResponse{
		HeadSlot:                 headState.Slot,
		BeaconBlockRootHash32:    headRoot[:],
		EpochBoundaryRootHash32:  epochBoundaryRoot,
		JustifiedEpoch:           headState.JustifiedEpoch,
		JustifiedBlockRootHash32: justifiedBlockRoot,
		LatestCrosslink:          headState.LatestCrosslinks[req.Shard],
	}
	if err := as.cache.Put(ctx, req, res); err != nil {
		return nil, err
	}
	return res, nil
}
