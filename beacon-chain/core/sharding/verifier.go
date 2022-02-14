package sharding

import (
	"bytes"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
)

func verifyIntermediateBlockBid(st state.BeaconState, blk block.BeaconBlock) error {
	if time.IsIntermediateBlockSlot(blk.Slot()) {
		return verifyIntermediateBlockBid(st, blk)
	}
	return verifyBidAtNormalBlockSlot(st, blk)
}

func verifyBidAtIntermediateBlockSlot(st state.BeaconState, blk block.BeaconBlock) error {
	// get last intermediate block from beacon state
	b := &ethpb.BeaconBlockDankSharding{}
	blockBid := b.Body.PayloadData.GetBlockBid()
	// Verify block in state contains bid (selector should be 0)
	if b.Slot+1 != blk.Slot() {
		return fmt.Errorf("intermediate block slot %d +1 does not match slot %d", b.Slot+1, blk.Slot())
	}
	// Verify intermediate block does not contain bid (selector should be 1)
	blockData := &ethpb.IntermediateBlockData{}

	blockDataPayloadRoot, err := blockData.ExecutionPayload.HashTreeRoot()
	if err != nil {
		return err
	}
	if blockDataPayloadRoot != bytesutil.ToBytes32(blockBid.ExecutionPayloadRoot) {
		return fmt.Errorf("intermediate block data root %#x does not match block data root %#x", blockDataPayloadRoot, bytesutil.ToBytes32(blockBid.ExecutionPayloadRoot))
	}
	if blockBid.ShardedDataCommitmentCount != blockData.ShardedCommitmentsContainer.IncludedShardedDataCommitments {
		return fmt.Errorf("intermediate block sharded data commitment count %d does not match block sharded data commitment count %d", blockBid.ShardedDataCommitmentCount, blockData.ShardedCommitmentsContainer.IncludedShardedDataCommitments)
	}
	cr := blockBid.ShardedDataCommitmentRoot
	l := uint64(len(blockData.ShardedCommitmentsContainer.ShardedCommitments)) - blockData.ShardedCommitmentsContainer.IncludedShardedDataCommitments
	_ = blockData.ShardedCommitmentsContainer.ShardedCommitments[l:]
	// HTR of sharded commitments HTR(shardedCommitments)
	if bytesutil.ToBytes32(cr) != [32]byte{} {
		return fmt.Errorf("intermediate block sharded data commitment root %#x does not match block sharded data commitment root %#x", cr, [32]byte{})
	}
	if blockBid.ValidatorIndex != blk.ProposerIndex() {
		return fmt.Errorf("intermediate block proposer index %d does not match block proposer index %d", blockBid.ValidatorIndex, blk.ProposerIndex())
	}
	return nil
}

func verifyBidAtNormalBlockSlot(st state.BeaconState, blk block.BeaconBlock) error {
	// Verify payload data contains bid (selector should be 0)
	blockBid := &ethpb.IntermediateBlockBid{}
	if blockBid.Slot != blk.Slot() {
		return fmt.Errorf("intermediate block slot %d does not match slot %d", blockBid.Slot, blk.Slot())
	}
	if bytes.Equal(blockBid.ParentBlockRoot, blk.ParentRoot()) {
		return fmt.Errorf("intermediate block parent block root %#x does not match block parent block root %#x", blockBid.ParentBlockRoot, blk.ParentRoot())
	}
	// We do not check that the builder address exists or has sufficient balance here.
	//        # If it does not have sufficient balance, the block proposer loses out, so it is their
	//        # responsibility to check.
	return nil
}
