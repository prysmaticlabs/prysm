// Package state implements the whole state transition
// function which consists of per slot, per-epoch transitions.
// It also bootstraps the genesis beacon state for slot 0.
package state

import (
	"context"
	"fmt"

	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	e "github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var log = logrus.WithField("prefix", "core/state")

// TransitionConfig defines important configuration options
// for executing a state transition, which can have logging and signature
// verification on or off depending on when and where it is used.
type TransitionConfig struct {
	VerifySignatures bool
	Logging          bool
}

// DefaultConfig option for executing state transitions.
func DefaultConfig() *TransitionConfig {
	return &TransitionConfig{
		VerifySignatures: false,
		Logging:          false,
	}
}

// ExecuteStateTransition defines the procedure for a state transition function.
// Spec pseudocode definition:
//  We now define the state transition function. At a high level the state transition is made up of three parts:
//  - The per-slot transitions, which happens at the start of every slot.
//  - The per-block transitions, which happens at every block.
//  - The per-epoch transitions, which happens at the end of the last slot of every epoch (i.e. (state.slot + 1) % SLOTS_PER_EPOCH == 0).
//  The per-slot transitions focus on the slot counter and block roots records updates.
//  The per-block transitions focus on verifying aggregate signatures and saving temporary records relating to the per-block activity in the state.
//  The per-epoch transitions focus on the validator registry, including adjusting balances and activating and exiting validators,
//  as well as processing crosslinks and managing block justification/finalization.
func ExecuteStateTransition(
	ctx context.Context,
	state *pb.BeaconState,
	block *pb.BeaconBlock,
	headRoot [32]byte,
	config *TransitionConfig,
) (*pb.BeaconState, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.StateTransition")
	defer span.End()
	var err error

	// Execute per slot transition.
	state = ProcessSlot(ctx, state, headRoot)

	// Execute per block transition.
	if block != nil {
		state, err = ProcessBlock(ctx, state, block, config)
		if err != nil {
			return nil, fmt.Errorf("could not process block: %v", err)
		}
	}

	// Execute per epoch transition.
	if e.CanProcessEpoch(state) {
		state, err = ProcessEpoch(ctx, state, block, config)
	}
	if err != nil {
		return nil, fmt.Errorf("could not process epoch: %v", err)
	}

	return state, nil
}

// ProcessSlot happens every slot and focuses on the slot counter and block roots record updates.
// It happens regardless if there's an incoming block or not.
//
// Spec pseudocode definition:
//	Set state.slot += 1
//	Let previous_block_root be the hash_tree_root of the previous beacon block processed in the chain
//	Set state.latest_block_roots[(state.slot - 1) % LATEST_BLOCK_ROOTS_LENGTH] = previous_block_root
//	If state.slot % LATEST_BLOCK_ROOTS_LENGTH == 0
//		append merkle_root(state.latest_block_roots) to state.batched_block_roots
func ProcessSlot(ctx context.Context, state *pb.BeaconState, headRoot [32]byte) *pb.BeaconState {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.state.ProcessSlot")
	defer span.End()
	state.Slot++
	state = b.ProcessBlockRoots(state, headRoot)
	return state
}

// ProcessBlock creates a new, modified beacon state by applying block operation
// transformations as defined in the Ethereum Serenity specification, including processing proposer slashings,
// processing block attestations, and more.
func ProcessBlock(
	ctx context.Context,
	state *pb.BeaconState,
	block *pb.BeaconBlock,
	config *TransitionConfig,
) (*pb.BeaconState, error) {

	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.state.ProcessBlock")
	defer span.End()

	r, err := hashutil.HashBeaconBlock(block)
	if err != nil {
		return nil, fmt.Errorf("could not hash block: %v", err)
	}

	// Below are the processing steps to verify every block.
	// Verify block slot.
	if block.Slot != state.Slot {
		return nil, fmt.Errorf(
			"block.slot != state.slot, block.slot = %d, state.slot = %d",
			block.Slot,
			state.Slot,
		)
	}

	// Verify block signature.
	if config.VerifySignatures {
		// TODO(#781): Verify Proposer Signature.
		if err := b.VerifyProposerSignature(block); err != nil {
			return nil, fmt.Errorf("could not verify proposer signature: %v", err)
		}
	}

	// Save latest block.
	state.LatestBlock = block

	// Verify block RANDAO.
	state, err = b.ProcessRandao(state, block.Body, config.VerifySignatures, config.Logging)
	if err != nil {
		return nil, fmt.Errorf("could not verify and process randao: %v", err)
	}

	// Process ETH1 data.
	state = b.ProcessEth1DataInBlock(state, block)
	state, err = b.ProcessAttesterSlashings(state, block, config.VerifySignatures)
	if err != nil {
		return nil, fmt.Errorf("could not verify block attester slashings: %v", err)
	}

	state, err = b.ProcessProposerSlashings(state, block, config.VerifySignatures)
	if err != nil {
		return nil, fmt.Errorf("could not verify block proposer slashings: %v", err)
	}

	state, err = b.ProcessBlockAttestations(state, block, config.VerifySignatures)
	if err != nil {
		return nil, fmt.Errorf("could not process block attestations: %v", err)
	}

	state, err = b.ProcessValidatorDeposits(state, block, config.VerifySignatures)
	if err != nil {
		return nil, fmt.Errorf("could not process block validator deposits: %v", err)
	}
	state, err = b.ProcessValidatorExits(state, block, config.VerifySignatures)
	if err != nil {
		return nil, fmt.Errorf("could not process validator exits: %v", err)
	}

	if config.Logging {
		log.WithField("blockRoot", fmt.Sprintf("%#x", bytesutil.Trunc(r[:]))).Debugf("Verified block slot == state slot")
		log.WithField("blockRoot", fmt.Sprintf("%#x", bytesutil.Trunc(r[:]))).Debugf("Verified and processed block RANDAO")
		log.WithField("blockRoot", fmt.Sprintf("%#x", bytesutil.Trunc(r[:]))).Debugf("Processed ETH1 data")
		log.WithField(
			"attestationsInBlock", len(block.Body.Attestations),
		).Info("Block attestations")
		log.WithField(
			"depositsInBlock", len(block.Body.Deposits),
		).Info("Block deposits")
	}
	return state, nil
}

// ProcessEpoch describes the per epoch operations that are performed on the
// beacon state.
//
func ProcessEpoch(ctx context.Context, state *pb.BeaconState, _ *pb.BeaconBlock, _ *TransitionConfig) (*pb.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.state.ProcessEpoch")
	defer span.End()

	// TODO(#2307): Implement process epoch based on 0.6.

	return state, nil
}
