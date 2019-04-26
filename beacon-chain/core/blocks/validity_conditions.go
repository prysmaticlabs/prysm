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
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "core/blocks")

// IsValidBlock ensures that the block is compliant with the block processing validity conditions.
// Spec:
//  For a beacon chain block, block, to be processed by a node, the following conditions must be met:
//  The parent block with root block.parent_root has been processed and accepted.
//  The node has processed its state up to slot, block.slot - 1.
//  The Ethereum 1.0 block pointed to by the state.processed_pow_receipt_root has been processed and accepted.
//  The node's local clock time is greater than or equal to state.genesis_time + block.slot * SECONDS_PER_SLOT.
func IsValidBlock(
	ctx context.Context,
	state *pb.BeaconState,
	block *pb.BeaconBlock,
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

	h := common.BytesToHash(state.LatestEth1Data.BlockHash32)
	powBlock, err := GetPOWBlock(ctx, h)
	if err != nil {
		return fmt.Errorf("unable to retrieve POW chain reference block: %v", err)
	}

	// Pre-Processing Condition 2:
	// The block pointed to by the state in state.processed_pow_receipt_root has
	// been processed in the ETH 1.0 chain.
	if powBlock == nil {
		return fmt.Errorf("proof-of-Work chain reference in state does not exist: %#x", state.LatestEth1Data.BlockHash32)
	}

	// Pre-Processing Condition 4:
	// The node's local time is greater than or equal to
	// state.genesis_time + (block.slot-GENESIS_SLOT)* SECONDS_PER_SLOT.
	if !IsSlotValid(block.Slot, genesisTime) {
		return fmt.Errorf("slot of block is too high: %d", block.Slot-params.BeaconConfig().GenesisSlot)
	}

	return nil
}

// IsSlotValid compares the slot to the system clock to determine if the block is valid.
func IsSlotValid(slot uint64, genesisTime time.Time) bool {
	secondsPerSlot := time.Duration((slot-params.BeaconConfig().GenesisSlot)*params.BeaconConfig().SecondsPerSlot) * time.Second
	validTimeThreshold := genesisTime.Add(secondsPerSlot)
	now := clock.Now()
	isValid := now.After(validTimeThreshold)
	if !isValid {
		log.WithFields(logrus.Fields{
			"localTime":           now,
			"genesisPlusSlotTime": validTimeThreshold,
		}).Info("Waiting for slot to be valid")
	}
	return isValid
}
