package state

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/types"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
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
	isInChain func(blockHash [32]byte) bool,
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

	if err := doesParentProposerExist(block, beaconState, parentSlot); err != nil {
		return fmt.Errorf("could not get proposer index: %v", err)
	}

	for _, attestation := range block.Attestations() {
		if err := isBlockAttestationValid(block, attestation, beaconState, parentSlot, isInChain); err != nil {
			return fmt.Errorf("invalid block attestation: %v", err)
		}
	}

	_, proposerIndex, err := v.ProposerShardAndIndex(
		beaconState.ShardAndCommitteesForSlots(),
		beaconState.LastStateRecalculationSlot(),
		block.SlotNumber(),
	)
	if err != nil {
		return fmt.Errorf("could not get proposer index: %v", err)
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

// doesParentProposerExist checks that the proposer from the parent slot is included in the first
// aggregated attestation object
func doesParentProposerExist(block *types.Block, beaconState *types.BeaconState, parentSlot uint64) error {
	_, parentProposerIndex, err := v.ProposerShardAndIndex(
		beaconState.ShardAndCommitteesForSlots(),
		beaconState.LastStateRecalculationSlot(),
		parentSlot,
	)
	if err != nil {
		return err
	}

	// Verifies the attester bitfield to check if the proposer index is in the first included one.
	if isBitSet, err := bitutil.CheckBit(block.Attestations()[0].AttesterBitfield, int(parentProposerIndex)); !isBitSet {
		return fmt.Errorf("could not locate proposer in the first attestation of AttestionRecord: %v", err)
	}
	return nil
}

// isBlockAttestationValid verifies a block's attestations pass validity conditions.
func isBlockAttestationValid(
	block *types.Block,
	attestation *pb.AggregatedAttestation,
	beaconState *types.BeaconState,
	parentSlot uint64,
	isInChain func(blockHash [32]byte) bool,
) error {
	// Validate attestation's slot number has is within range of incoming block number.
	if err := isAttestationSlotNumberValid(attestation.Slot, parentSlot); err != nil {
		return fmt.Errorf("invalid attestation slot %d: %v", attestation.Slot, err)
	}

	if attestation.JustifiedSlot > beaconState.LastJustifiedSlot() {
		return fmt.Errorf(
			"attestation's justified slot has to be <= the state's last justified slot: found: %d. want <=: %d",
			attestation.JustifiedSlot,
			beaconState.LastJustifiedSlot(),
		)
	}

	hash := [32]byte{}
	copy(hash[:], attestation.JustifiedBlockHash)
	if !isInChain(hash) {
		return fmt.Errorf(
			"the attestation's justifed block hash not found in current chain: justified block hash: 0x%x",
			attestation.JustifiedBlockHash,
		)
	}

	// Get all the block hashes up to cycle length.
	parentHashes, err := beaconState.SignedParentHashes(block, attestation)
	if err != nil {
		return fmt.Errorf("unable to get signed parent hashes: %v", err)
	}

	shardCommittees, err := v.GetShardAndCommitteesForSlot(
		beaconState.ShardAndCommitteesForSlots(),
		beaconState.LastStateRecalculationSlot(),
		attestation.Slot,
	)
	attesterIndices, err := v.AttesterIndices(shardCommittees, attestation)
	if err != nil {
		return fmt.Errorf("unable to get validator committee: %v", err)
	}

	// Verify attester bitfields matches crystallized state's prev computed bitfield.
	if !v.AreAttesterBitfieldsValid(attestation, attesterIndices) {
		return fmt.Errorf("Unable to match attester bitfield with shard and committee bitfield")
	}

	forkVersion := beaconState.PostForkVersion()
	if attestation.Slot < beaconState.ForkSlotNumber() {
		forkVersion = beaconState.PreForkVersion()
	}

	// TODO(#258): Generate validators aggregated pub key.
	attestationMsg := types.AttestationMsg(
		parentHashes,
		attestation.ShardBlockHash,
		attestation.Slot,
		attestation.Shard,
		attestation.JustifiedSlot,
		forkVersion,
	)
	_ = attestationMsg

	// TODO(#258): Verify msgHash against aggregated pub key and aggregated signature.
	return nil
}

func isAttestationSlotNumberValid(attestationSlot uint64, parentSlot uint64) error {
	if parentSlot != 0 && attestationSlot > parentSlot {
		return fmt.Errorf(
			"attestation slot number higher than parent block's slot number: found: %d, needed < %d",
			attestationSlot,
			parentSlot,
		)
	}
	if parentSlot >= params.BeaconConfig().CycleLength-1 && attestationSlot < parentSlot-params.BeaconConfig().CycleLength+1 {
		return fmt.Errorf(
			"attestation slot number lower than parent block's slot number by one CycleLength: found: %d, needed > %d",
			attestationSlot,
			parentSlot-params.BeaconConfig().CycleLength+1,
		)
	}
	return nil
}
