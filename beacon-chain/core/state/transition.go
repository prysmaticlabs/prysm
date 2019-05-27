// Package state implements the whole state transition
// function which consists of per slot, per-epoch transitions.
// It also bootstraps the genesis beacon state for slot 0.
package state

import (
	"bytes"
	"context"
	"fmt"

	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	e "github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/ssz"
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
//  - The per-slot transitions focuses on increasing the slot number and recording recent block headers.
//  - The per-epoch transitions focuses on the validator registry, adjusting balances, and finalizing slots.
//  - The per-block transitions focuses on verifying block operations, verifying attestations, and signatures.
func ExecuteStateTransition(
	ctx context.Context,
	state *pb.BeaconState,
	block *pb.BeaconBlock,
	config *TransitionConfig,
) (*pb.BeaconState, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.StateTransition")
	defer span.End()
	var err error

	// Execute per slot transition.
	if state.Slot >= block.Slot {
		return nil, fmt.Errorf("expected state.slot %d < block.slot %d", state.Slot, block.Slot)
	}
	for state.Slot < block.Slot {
		state, err = ProcessSlot(ctx, state)
		if err != nil {
			return nil, fmt.Errorf("could not process slot: %v", err)
		}
		if e.CanProcessEpoch(state) {
			state, err = ProcessEpoch(ctx, state)
			if err != nil {
				return nil, fmt.Errorf("could not process epoch: %v", err)
			}
		}
		state.Slot++
	}

	// Execute per block transition.
	if block != nil {
		state, err = ProcessBlock(ctx, state, block, config)
		if err != nil {
			return nil, fmt.Errorf("could not process block: %v", err)
		}
	}
	// TODO(#2307): Validate state root.
	return state, nil
}

// ProcessSlot happens every slot and focuses on the slot counter and block roots record updates.
// It happens regardless if there's an incoming block or not.
func ProcessSlot(ctx context.Context, state *pb.BeaconState) (*pb.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.state.ProcessSlot")
	defer span.End()
	prevStateRoot, err := ssz.TreeHash(state)
	if err != nil {
		return nil, fmt.Errorf("could not tree hash prev state root: %v", err)
	}
	state.LatestStateRoots[state.Slot%params.BeaconConfig().SlotsPerHistoricalRoot] = prevStateRoot[:]
	zeroHash := params.BeaconConfig().ZeroHash

	// Cache latest block header state root.
	if bytes.Equal(state.LatestBlockHeader.StateRoot, zeroHash[:]) {
		state.LatestBlockHeader.StateRoot = prevStateRoot[:]
	}
	prevBlockRoot, err := ssz.SigningRoot(state.LatestBlockHeader)
	if err != nil {
		return nil, fmt.Errorf("could not determine prev block root: %v", err)
	}
	// Cache the block root.
	state.LatestBlockRoots[state.Slot%params.BeaconConfig().SlotsPerHistoricalRoot] = prevBlockRoot[:]
	return state, nil
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

	// Save latest block.
	state.LatestBlock = block

	// Process the block's header into the state.
	state, err = b.ProcessBlockHeader(state, block)
	if err != nil {
		return nil, fmt.Errorf("could not process block header: %v", err)
	}
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
	state, err = b.ProcessTransfers(state, block, config.VerifySignatures)
	if err != nil {
		return nil, fmt.Errorf("could not process block transfers: %v", err)
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
// beacon state. It focuses on the validator registry, adjusting balances, and finalizing slots.
// Spec pseudocode definition:
//
//  def process_epoch(state: BeaconState) -> None:
//    process_justification_and_finalization(state)
//    process_crosslinks(state)
//    process_rewards_and_penalties(state)
//    process_registry_updates(state)
//    process_slashings(state)
//    process_final_updates(state)
func ProcessEpoch(ctx context.Context, state *pb.BeaconState) (*pb.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.state.ProcessEpoch")
	defer span.End()

	prevEpochAtts, err := e.MatchAttestations(state, helpers.PrevEpoch(state))
	if err != nil {
		return nil, fmt.Errorf("could not get target atts prev epoch %d: %v",
			helpers.PrevEpoch(state), err)
	}
	currentEpochAtts, err := e.MatchAttestations(state, helpers.CurrentEpoch(state))
	if err != nil {
		return nil, fmt.Errorf("could not get target atts current epoch %d: %v",
			helpers.CurrentEpoch(state), err)
	}
	prevEpochAttestedBalance, err := e.AttestingBalance(state, prevEpochAtts.Target)
	if err != nil {
		return nil, fmt.Errorf("could not get attesting balance prev epoch: %v", err)
	}
	currentEpochAttestedBalance, err := e.AttestingBalance(state, currentEpochAtts.Target)
	if err != nil {
		return nil, fmt.Errorf("could not get attesting balance current epoch: %v", err)
	}

	state, err = e.ProcessJustificationAndFinalization(state, prevEpochAttestedBalance, currentEpochAttestedBalance)
	if err != nil {
		return nil, fmt.Errorf("could not process justification: %v", err)
	}

	state, err = e.ProcessCrosslink(state)
	if err != nil {
		return nil, fmt.Errorf("could not process crosslink: %v", err)
	}

	state, err = e.ProcessRewardsAndPenalties(state)
	if err != nil {
		return nil, fmt.Errorf("could not process rewards and penalties: %v", err)
	}

	state = e.ProcessRegistryUpdates(state)

	state = e.ProcessSlashings(state)

	state, err = e.ProcessFinalUpdates(state)
	if err != nil {
		return nil, fmt.Errorf("could not process final updates: %v", err)
	}

	return state, nil
}
