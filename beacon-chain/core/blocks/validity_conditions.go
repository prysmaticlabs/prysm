// Package blocks contains block processing libraries. These libraries
// process and verify block specific messages such as PoW receipt root,
// RANDAO, validator deposits, exits and slashing proofs.
package blocks

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	bytesutil "github.com/prysmaticlabs/prysm/shared/bytes"
)

// IsValidBlock ensures that the block is compliant with the block processing validity conditions.
// Spec:
//  For a beacon chain block, block, to be processed by a node, the following conditions must be met:
//  The parent block with root block.parent_root has been processed and accepted.
//  The node has processed its state up to slot, block.slot - 1.
//  The Ethereum 1.0 block pointed to by the state.processed_pow_receipt_root has been processed and accepted.
//  The node's local clock time is greater than or equal to state.genesis_time + block.slot * SLOT_DURATION.
func IsValidBlock(
	ctx context.Context,
	state *pb.BeaconState,
	block *pb.BeaconBlock,
	enablePOWChain bool,
	HasBlock func(hash [32]byte) bool,
	GetPOWBlock func(ctx context.Context, hash common.Hash) (*gethTypes.Block, error),
	genesisTime time.Time) error {

	// Pre-Processing Condition 1:
	// Check that the parent Block has been processed and saved.
	parentRoot := bytesutil.ToBytes32(block.ParentRootHash32)
	parentBlock := HasBlock(parentRoot)
	if !parentBlock {
		return fmt.Errorf("unprocessed parent block as it is not saved in the db: %#x", parentRoot)
	}

	// Pre-Processing Condition 2:
	// The state is updated up to block.slot -1.

	if state.Slot != block.Slot-1 {
		return fmt.Errorf(
			"block slot is not valid %d as it is supposed to be %d", block.Slot, state.Slot+1)
	}

	if enablePOWChain {
		h := common.BytesToHash(state.LatestDepositRootHash32)
		powBlock, err := GetPOWBlock(ctx, h)
		if err != nil {
			return fmt.Errorf("unable to retrieve POW chain reference block %v", err)
		}

		// Pre-Processing Condition 3:
		// The block pointed to by the state in state.processed_pow_receipt_root has
		// been processed in the ETH 1.0 chain.
		if powBlock == nil {
			return fmt.Errorf("proof-of-Work chain reference in state does not exist %#x", state.LatestDepositRootHash32)
		}
	}

	// Pre-Processing Condition 4:
	// The node's local time is greater than or equal to
	// state.genesis_time + block.slot * SLOT_DURATION.
	if !IsSlotValid(block.Slot, genesisTime) {
		return fmt.Errorf("slot of block is too high: %d", block.Slot)
	}

	return nil
}
