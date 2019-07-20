package blocks

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// IsValidBlock ensures that the block is compliant with the block processing validity conditions.
func IsValidBlock(
	ctx context.Context,
	state *pb.BeaconState,
	block *pb.BeaconBlock,
	HasBlock func(hash [32]byte) bool,
	GetPOWBlock func(ctx context.Context, hash common.Hash) (*gethTypes.Block, error),
	genesisTime time.Time) error {

	// Pre-Processing Condition 1:
	// Check that the parent Block has been processed and saved.
	parentRoot := bytesutil.ToBytes32(block.ParentRoot)
	parentBlock := HasBlock(parentRoot)
	if !parentBlock {
		return fmt.Errorf("unprocessed parent block as it is not saved in the db: %#x", parentRoot)
	}

	h := common.BytesToHash(state.Eth1Data.BlockHash)
	powBlock, err := GetPOWBlock(ctx, h)
	if err != nil {
		return fmt.Errorf("unable to retrieve POW chain reference block: %v", err)
	}

	// Pre-Processing Condition 2:
	// The block pointed to by the state in state.processed_pow_receipt_root has
	// been processed in the ETH 1.0 chain.
	if powBlock == nil {
		return fmt.Errorf("proof-of-Work chain reference in state does not exist: %#x", state.Eth1Data.BlockHash)
	}

	// Pre-Processing Condition 4:
	// The node's local time is greater than or equal to
	// state.genesis_time + (block.slot-GENESIS_SLOT)* SECONDS_PER_SLOT.
	if !IsSlotValid(block.Slot, genesisTime) {
		return fmt.Errorf("slot of block is too high: %d", block.Slot)
	}

	return nil
}

// IsSlotValid compares the slot to the system clock to determine if the block is valid.
func IsSlotValid(slot uint64, genesisTime time.Time) bool {
	secondsPerSlot := time.Duration((slot)*params.BeaconConfig().SecondsPerSlot) * time.Second
	validTimeThreshold := genesisTime.Add(secondsPerSlot)
	now := clock.Now()
	isValid := now.After(validTimeThreshold)

	return isValid
}
