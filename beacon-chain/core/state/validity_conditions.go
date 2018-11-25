package state

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/types"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// IsValidBlock verifies a block is valid according to the ETH 2.0 specification for
// validity conditions taking into consideration attestation processing and more.
func IsValidBlock(
	block *types.Block,
	beaconState *types.BeaconState,
	parentSlot uint64,
	genesisTime time.Time,
) error {
	_, err := block.Hash()
	if err != nil {
		return fmt.Errorf("could not hash incoming block: %v", err)
	}

	if block.SlotNumber() == 0 {
		return errors.New("cannot process a genesis block: received block with slot 0")
	}

	if !block.IsSlotValid(genesisTime) {
		return fmt.Errorf("slot of block is too high: %d", block.SlotNumber())
	}

	// if !b.doesParentProposerExist(cState, parentSlot) || !b.areAttestationsValid(db, aState, cState, parentSlot) {
	// 	log.Error("Invalid attestation")
	// 	return false
	// }

	_, proposerIndex, err := v.ProposerShardAndIndex(
		beaconState.ShardAndCommitteesForSlots(),
		beaconState.LastStateRecalculationSlot(),
		block.SlotNumber(),
	)
	if err != nil {
		return fmt.Errorf("Could not get proposer index: %v", err)
	}

	stateProposerRandaoSeed := beaconState.Validators()[proposerIndex].RandaoCommitment
	blockRandaoReveal := block.RandaoReveal()

	// If this is a block created by the simulator service (while in development
	// mode), we skip the RANDAO validation condition.
	isSimulatedBlock := bytes.Equal(blockRandaoReveal[:], params.BeaconConfig().SimulatedBlockRandao[:])
	if !isSimulatedBlock && !block.IsRandaoValid(stateProposerRandaoSeed) {
		return fmt.Errorf("Pre-image of %#x is %#x, Got: %#x", blockRandaoReveal[:], hashutil.Hash(blockRandaoReveal[:]), stateProposerRandaoSeed)
	}
	return nil
}
