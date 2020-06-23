// Package state implements the whole state transition
// function which consists of per slot, per-epoch transitions.
// It also bootstraps the genesis beacon state for slot 0.
package state

import (
	"bytes"
	"context"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	e "github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch/precompute"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/interop"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// ExecuteStateTransition defines the procedure for a state transition function.
//
// Spec pseudocode definition:
//  def state_transition(state: BeaconState, signed_block: SignedBeaconBlock, validate_result: bool=True) -> BeaconState:
//    block = signed_block.message
//    # Process slots (including those with no blocks) since block
//    process_slots(state, block.slot)
//    # Verify signature
//    if validate_result:
//        assert verify_block_signature(state, signed_block)
//    # Process block
//    process_block(state, block)
//    # Verify state root
//    if validate_result:
//        assert block.state_root == hash_tree_root(state)
//    # Return post-state
//    return state
func ExecuteStateTransition(
	ctx context.Context,
	state *stateTrie.BeaconState,
	signed *ethpb.SignedBeaconBlock,
) (*stateTrie.BeaconState, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if signed == nil || signed.Block == nil {
		return nil, errors.New("nil block")
	}

	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.ExecuteStateTransition")
	defer span.End()
	var err error
	// Execute per slots transition.
	state, err = ProcessSlots(ctx, state, signed.Block.Slot)
	if err != nil {
		return nil, errors.Wrap(err, "could not process slot")
	}

	// Execute per block transition.
	state, err = ProcessBlock(ctx, state, signed)
	if err != nil {
		return nil, errors.Wrapf(err, "could not process block in slot %d", signed.Block.Slot)
	}

	interop.WriteBlockToDisk(signed, false)
	interop.WriteStateToDisk(state)

	postStateRoot, err := state.HashTreeRoot(ctx)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(postStateRoot[:], signed.Block.StateRoot) {
		return state, fmt.Errorf("validate state root failed, wanted: %#x, received: %#x",
			postStateRoot[:], signed.Block.StateRoot)
	}
	return state, nil
}

// ExecuteStateTransitionNoVerifyAttSigs defines the procedure for a state transition function.
// This does not validate any BLS signatures of attestations in a block, it is used for performing a state transition as quickly
// as possible. This function should only be used when we can trust the data we're receiving entirely, such as
// initial sync or for processing past accepted blocks.
//
// WARNING: This method does not validate any signatures in a block. This method also modifies the passed in state.
//
// Spec pseudocode definition:
//  def state_transition(state: BeaconState, block: BeaconBlock, validate_state_root: bool=False) -> BeaconState:
//    # Process slots (including those with no blocks) since block
//    process_slots(state, block.slot)
//    # Process block
//    process_block(state, block)
//    # Return post-state
//    return state
func ExecuteStateTransitionNoVerifyAttSigs(
	ctx context.Context,
	state *stateTrie.BeaconState,
	signed *ethpb.SignedBeaconBlock,
) (*stateTrie.BeaconState, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if signed == nil || signed.Block == nil {
		return nil, errors.New("nil block")
	}

	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.ExecuteStateTransitionNoVerifyAttSigs")
	defer span.End()
	var err error

	// Execute per slots transition.
	state, err = ProcessSlots(ctx, state, signed.Block.Slot)
	if err != nil {
		return nil, errors.Wrap(err, "could not process slot")
	}

	// Execute per block transition.
	state, err = ProcessBlockNoVerifyAttSigs(ctx, state, signed)
	if err != nil {
		return nil, errors.Wrap(err, "could not process block")
	}

	return state, nil
}

// CalculateStateRoot defines the procedure for a state transition function.
// This does not validate any BLS signatures in a block, it is used for calculating the
// state root of the state for the block proposer to use.
// This does not modify state.
//
// WARNING: This method does not validate any BLS signatures. This is used for proposer to compute
// state root before proposing a new block, and this does not modify state.
//
// Spec pseudocode definition:
//  def state_transition(state: BeaconState, block: BeaconBlock, validate_state_root: bool=False) -> BeaconState:
//    # Process slots (including those with no blocks) since block
//    process_slots(state, block.slot)
//    # Process block
//    process_block(state, block)
//    # Return post-state
//    return state
func CalculateStateRoot(
	ctx context.Context,
	state *stateTrie.BeaconState,
	signed *ethpb.SignedBeaconBlock,
) ([32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.CalculateStateRoot")
	defer span.End()
	if ctx.Err() != nil {
		traceutil.AnnotateError(span, ctx.Err())
		return [32]byte{}, ctx.Err()
	}
	if state == nil {
		return [32]byte{}, errors.New("nil state")
	}
	if signed == nil || signed.Block == nil {
		return [32]byte{}, errors.New("nil block")
	}

	// Copy state to avoid mutating the state reference.
	state = state.Copy()

	// Execute per slots transition.
	state, err := ProcessSlots(ctx, state, signed.Block.Slot)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not process slot")
	}

	// Execute per block transition.
	state, err = ProcessBlockForStateRoot(ctx, state, signed)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not process block")
	}

	return state.HashTreeRoot(ctx)
}

// ProcessSlot happens every slot and focuses on the slot counter and block roots record updates.
// It happens regardless if there's an incoming block or not.
// Spec pseudocode definition:
//
//  def process_slot(state: BeaconState) -> None:
//    # Cache state root
//    previous_state_root = hash_tree_root(state)
//    state.state_roots[state.slot % SLOTS_PER_HISTORICAL_ROOT] = previous_state_root
//
//    # Cache latest block header state root
//    if state.latest_block_header.state_root == Bytes32():
//        state.latest_block_header.state_root = previous_state_root
//
//    # Cache block root
//    previous_block_root = hash_tree_root(state.latest_block_header)
//    state.block_roots[state.slot % SLOTS_PER_HISTORICAL_ROOT] = previous_block_root
func ProcessSlot(ctx context.Context, state *stateTrie.BeaconState) (*stateTrie.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.state.ProcessSlot")
	defer span.End()
	span.AddAttributes(trace.Int64Attribute("slot", int64(state.Slot())))

	prevStateRoot, err := state.HashTreeRoot(ctx)
	if err != nil {
		return nil, err
	}
	if err := state.UpdateStateRootAtIndex(
		state.Slot()%params.BeaconConfig().SlotsPerHistoricalRoot,
		prevStateRoot,
	); err != nil {
		return nil, err
	}

	zeroHash := params.BeaconConfig().ZeroHash
	// Cache latest block header state root.
	header := state.LatestBlockHeader()
	if header.StateRoot == nil || bytes.Equal(header.StateRoot, zeroHash[:]) {
		header.StateRoot = prevStateRoot[:]
		if err := state.SetLatestBlockHeader(header); err != nil {
			return nil, err
		}
	}
	prevBlockRoot, err := stateutil.BlockHeaderRoot(state.LatestBlockHeader())
	if err != nil {
		traceutil.AnnotateError(span, err)
		return nil, errors.Wrap(err, "could not determine prev block root")
	}
	// Cache the block root.
	if err := state.UpdateBlockRootAtIndex(
		state.Slot()%params.BeaconConfig().SlotsPerHistoricalRoot,
		prevBlockRoot,
	); err != nil {
		return nil, err
	}
	return state, nil
}

// ProcessSlots process through skip slots and apply epoch transition when it's needed
//
// Spec pseudocode definition:
//  def process_slots(state: BeaconState, slot: Slot) -> None:
//    assert state.slot <= slot
//    while state.slot < slot:
//        process_slot(state)
//        # Process epoch on the first slot of the next epoch
//        if (state.slot + 1) % SLOTS_PER_EPOCH == 0:
//            process_epoch(state)
//        state.slot += 1
//    ]
func ProcessSlots(ctx context.Context, state *stateTrie.BeaconState, slot uint64) (*stateTrie.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.ProcessSlots")
	defer span.End()
	if state == nil {
		return nil, errors.New("nil state")
	}
	span.AddAttributes(trace.Int64Attribute("slots", int64(slot)-int64(state.Slot())))

	// The block must have a higher slot than parent state.
	if state.Slot() >= slot {
		err := fmt.Errorf("expected state.slot %d < slot %d", state.Slot(), slot)
		traceutil.AnnotateError(span, err)
		return nil, err
	}

	highestSlot := state.Slot()
	key := state.Slot()

	// Restart from cached value, if one exists.
	cachedState, err := SkipSlotCache.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	if cachedState != nil && cachedState.Slot() < slot {
		highestSlot = cachedState.Slot()
		state = cachedState
	}
	if err := SkipSlotCache.MarkInProgress(key); err == cache.ErrAlreadyInProgress {
		cachedState, err = SkipSlotCache.Get(ctx, key)
		if err != nil {
			return nil, err
		}
		if cachedState != nil && cachedState.Slot() < slot {
			highestSlot = cachedState.Slot()
			state = cachedState
		}
	} else if err != nil {
		return nil, err
	}
	defer func() {
		if err := SkipSlotCache.MarkNotInProgress(key); err != nil {
			traceutil.AnnotateError(span, err)
			logrus.WithError(err).Error("Failed to mark skip slot no longer in progress")
		}
	}()

	for state.Slot() < slot {
		if ctx.Err() != nil {
			traceutil.AnnotateError(span, ctx.Err())
			// Cache last best value.
			if highestSlot < state.Slot() {
				if err := SkipSlotCache.Put(ctx, key, state); err != nil {
					logrus.WithError(err).Error("Failed to put skip slot cache value")
				}
			}
			return nil, ctx.Err()
		}
		state, err = ProcessSlot(ctx, state)
		if err != nil {
			traceutil.AnnotateError(span, err)
			return nil, errors.Wrap(err, "could not process slot")
		}
		if CanProcessEpoch(state) {
			state, err = ProcessEpochPrecompute(ctx, state)
			if err != nil {
				traceutil.AnnotateError(span, err)
				return nil, errors.Wrap(err, "could not process epoch with optimizations")
			}
		}
		if err := state.SetSlot(state.Slot() + 1); err != nil {
			traceutil.AnnotateError(span, err)
			return nil, errors.Wrap(err, "failed to increment state slot")
		}
	}

	if highestSlot < state.Slot() {
		if err := SkipSlotCache.Put(ctx, key, state); err != nil {
			logrus.WithError(err).Error("Failed to put skip slot cache value")
			traceutil.AnnotateError(span, err)
		}
	}

	return state, nil
}

// ProcessBlock creates a new, modified beacon state by applying block operation
// transformations as defined in the Ethereum Serenity specification, including processing proposer slashings,
// processing block attestations, and more.
//
// Spec pseudocode definition:
//
//  def process_block(state: BeaconState, block: BeaconBlock) -> None:
//    process_block_header(state, block)
//    process_randao(state, block.body)
//    process_eth1_data(state, block.body)
//    process_operations(state, block.body)
func ProcessBlock(
	ctx context.Context,
	state *stateTrie.BeaconState,
	signed *ethpb.SignedBeaconBlock,
) (*stateTrie.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.state.ProcessBlock")
	defer span.End()

	state, err := b.ProcessBlockHeader(state, signed)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return nil, errors.Wrap(err, "could not process block header")
	}

	state, err = b.ProcessRandao(state, signed.Block.Body)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return nil, errors.Wrap(err, "could not verify and process randao")
	}

	state, err = b.ProcessEth1DataInBlock(state, signed.Block)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return nil, errors.Wrap(err, "could not process eth1 data")
	}

	state, err = ProcessOperations(ctx, state, signed.Block.Body)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return nil, errors.Wrap(err, "could not process block operation")
	}

	return state, nil
}

// ProcessBlockNoVerifyAttSigs creates a new, modified beacon state by applying block operation
// transformations as defined in the Ethereum Serenity specification. It does not validate
// block attestation signatures.
//
// Spec pseudocode definition:
//
//  def process_block(state: BeaconState, block: BeaconBlock) -> None:
//    process_block_header(state, block)
//    process_randao(state, block.body)
//    process_eth1_data(state, block.body)
//    process_operations(state, block.body)
func ProcessBlockNoVerifyAttSigs(
	ctx context.Context,
	state *stateTrie.BeaconState,
	signed *ethpb.SignedBeaconBlock,
) (*stateTrie.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.state.ProcessBlock")
	defer span.End()

	state, err := b.ProcessBlockHeader(state, signed)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return nil, errors.Wrap(err, "could not process block header")
	}

	state, err = b.ProcessRandao(state, signed.Block.Body)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return nil, errors.Wrap(err, "could not verify and process randao")
	}

	state, err = b.ProcessEth1DataInBlock(state, signed.Block)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return nil, errors.Wrap(err, "could not process eth1 data")
	}

	state, err = ProcessOperationsNoVerify(ctx, state, signed.Block.Body)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return nil, errors.Wrap(err, "could not process block operation")
	}

	return state, nil
}

// ProcessOperations processes the operations in the beacon block and updates beacon state
// with the operations in block.
//
// Spec pseudocode definition:
//
//  def process_operations(state: BeaconState, body: BeaconBlockBody) -> None:
//    # Verify that outstanding deposits are processed up to the maximum number of deposits
//    assert len(body.deposits) == min(MAX_DEPOSITS, state.eth1_data.deposit_count - state.eth1_deposit_index)
//    # Verify that there are no duplicate transfers
//    assert len(body.transfers) == len(set(body.transfers))
//
//    all_operations = (
//        (body.proposer_slashings, process_proposer_slashing),
//        (body.attester_slashings, process_attester_slashing),
//        (body.attestations, process_attestation),
//        (body.deposits, process_deposit),
//        (body.voluntary_exits, process_voluntary_exit),
//        (body.transfers, process_transfer),
//    )  # type: Sequence[Tuple[List, Callable]]
//    for operations, function in all_operations:
//        for operation in operations:
//            function(state, operation)
func ProcessOperations(
	ctx context.Context,
	state *stateTrie.BeaconState,
	body *ethpb.BeaconBlockBody) (*stateTrie.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.state.ProcessOperations")
	defer span.End()

	if err := verifyOperationLengths(state, body); err != nil {
		return nil, errors.Wrap(err, "could not verify operation lengths")
	}

	state, err := b.ProcessProposerSlashings(ctx, state, body)
	if err != nil {
		return nil, errors.Wrap(err, "could not process block proposer slashings")
	}
	state, err = b.ProcessAttesterSlashings(ctx, state, body)
	if err != nil {
		return nil, errors.Wrap(err, "could not process block attester slashings")
	}
	state, err = b.ProcessAttestationsNoVerify(ctx, state, body)
	if err != nil {
		return nil, errors.Wrap(err, "could not process block attestations")
	}
	if err := b.VerifyAttestations(ctx, state, body.Attestations); err != nil {
		return nil, errors.Wrap(err, "could not verify attestations")
	}
	state, err = b.ProcessDeposits(ctx, state, body)
	if err != nil {
		return nil, errors.Wrap(err, "could not process block validator deposits")
	}
	state, err = b.ProcessVoluntaryExits(ctx, state, body)
	if err != nil {
		return nil, errors.Wrap(err, "could not process validator exits")
	}

	return state, nil
}

// ProcessOperationsNoVerify processes the operations in the beacon block and updates beacon state
// with the operations in block. It does not verify attestation signatures or voluntary exit signatures.
//
// WARNING: This method does not verify attestation signatures or voluntary exit signatures.
// This is used to perform the block operations as fast as possible.
//
// Spec pseudocode definition:
//
//  def process_operations(state: BeaconState, body: BeaconBlockBody) -> None:
//    # Verify that outstanding deposits are processed up to the maximum number of deposits
//    assert len(body.deposits) == min(MAX_DEPOSITS, state.eth1_data.deposit_count - state.eth1_deposit_index)
//    # Verify that there are no duplicate transfers
//    assert len(body.transfers) == len(set(body.transfers))
//
//    all_operations = (
//        (body.proposer_slashings, process_proposer_slashing),
//        (body.attester_slashings, process_attester_slashing),
//        (body.attestations, process_attestation),
//        (body.deposits, process_deposit),
//        (body.voluntary_exits, process_voluntary_exit),
//        (body.transfers, process_transfer),
//    )  # type: Sequence[Tuple[List, Callable]]
//    for operations, function in all_operations:
//        for operation in operations:
//            function(state, operation)
func ProcessOperationsNoVerify(
	ctx context.Context,
	state *stateTrie.BeaconState,
	body *ethpb.BeaconBlockBody) (*stateTrie.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.state.ProcessOperations")
	defer span.End()

	if err := verifyOperationLengths(state, body); err != nil {
		return nil, errors.Wrap(err, "could not verify operation lengths")
	}

	state, err := b.ProcessProposerSlashings(ctx, state, body)
	if err != nil {
		return nil, errors.Wrap(err, "could not process block proposer slashings")
	}
	state, err = b.ProcessAttesterSlashings(ctx, state, body)
	if err != nil {
		return nil, errors.Wrap(err, "could not process block attester slashings")
	}
	state, err = b.ProcessAttestationsNoVerify(ctx, state, body)
	if err != nil {
		return nil, errors.Wrap(err, "could not process block attestations")
	}
	state, err = b.ProcessDeposits(ctx, state, body)
	if err != nil {
		return nil, errors.Wrap(err, "could not process block validator deposits")
	}
	state, err = b.ProcessVoluntaryExitsNoVerify(state, body)
	if err != nil {
		return nil, errors.Wrap(err, "could not process validator exits")
	}

	return state, nil
}

func verifyOperationLengths(state *stateTrie.BeaconState, body *ethpb.BeaconBlockBody) error {
	if uint64(len(body.ProposerSlashings)) > params.BeaconConfig().MaxProposerSlashings {
		return fmt.Errorf(
			"number of proposer slashings (%d) in block body exceeds allowed threshold of %d",
			len(body.ProposerSlashings),
			params.BeaconConfig().MaxProposerSlashings,
		)
	}

	if uint64(len(body.AttesterSlashings)) > params.BeaconConfig().MaxAttesterSlashings {
		return fmt.Errorf(
			"number of attester slashings (%d) in block body exceeds allowed threshold of %d",
			len(body.AttesterSlashings),
			params.BeaconConfig().MaxAttesterSlashings,
		)
	}

	if uint64(len(body.Attestations)) > params.BeaconConfig().MaxAttestations {
		return fmt.Errorf(
			"number of attestations (%d) in block body exceeds allowed threshold of %d",
			len(body.Attestations),
			params.BeaconConfig().MaxAttestations,
		)
	}

	if uint64(len(body.VoluntaryExits)) > params.BeaconConfig().MaxVoluntaryExits {
		return fmt.Errorf(
			"number of voluntary exits (%d) in block body exceeds allowed threshold of %d",
			len(body.VoluntaryExits),
			params.BeaconConfig().MaxVoluntaryExits,
		)
	}
	eth1Data := state.Eth1Data()
	if eth1Data == nil {
		return errors.New("nil eth1data in state")
	}
	if state.Eth1DepositIndex() > eth1Data.DepositCount {
		return fmt.Errorf("expected state.deposit_index %d <= eth1data.deposit_count %d", state.Eth1DepositIndex(), eth1Data.DepositCount)
	}
	maxDeposits := mathutil.Min(params.BeaconConfig().MaxDeposits, eth1Data.DepositCount-state.Eth1DepositIndex())
	// Verify outstanding deposits are processed up to max number of deposits
	if len(body.Deposits) != int(maxDeposits) {
		return fmt.Errorf("incorrect outstanding deposits in block body, wanted: %d, got: %d",
			maxDeposits, len(body.Deposits))
	}

	return nil
}

// CanProcessEpoch checks the eligibility to process epoch.
// The epoch can be processed at the end of the last slot of every epoch
//
// Spec pseudocode definition:
//    If (state.slot + 1) % SLOTS_PER_EPOCH == 0:
func CanProcessEpoch(state *stateTrie.BeaconState) bool {
	return (state.Slot()+1)%params.BeaconConfig().SlotsPerEpoch == 0
}

// ProcessEpochPrecompute describes the per epoch operations that are performed on the beacon state.
// It's optimized by pre computing validator attested info and epoch total/attested balances upfront.
func ProcessEpochPrecompute(ctx context.Context, state *stateTrie.BeaconState) (*stateTrie.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.state.ProcessEpoch")
	defer span.End()
	span.AddAttributes(trace.Int64Attribute("epoch", int64(helpers.CurrentEpoch(state))))

	if state == nil {
		return nil, errors.New("nil state")
	}
	vp, bp, err := precompute.New(ctx, state)
	if err != nil {
		return nil, err
	}
	vp, bp, err = precompute.ProcessAttestations(ctx, state, vp, bp)
	if err != nil {
		return nil, err
	}

	state, err = precompute.ProcessJustificationAndFinalizationPreCompute(state, bp)
	if err != nil {
		return nil, errors.Wrap(err, "could not process justification")
	}

	state, err = precompute.ProcessRewardsAndPenaltiesPrecompute(state, bp, vp)
	if err != nil {
		return nil, errors.Wrap(err, "could not process rewards and penalties")
	}

	state, err = e.ProcessRegistryUpdates(state)
	if err != nil {
		return nil, errors.Wrap(err, "could not process registry updates")
	}

	err = precompute.ProcessSlashingsPrecompute(state, bp)
	if err != nil {
		return nil, err
	}

	state, err = e.ProcessFinalUpdates(state)
	if err != nil {
		return nil, errors.Wrap(err, "could not process final updates")
	}
	return state, nil
}

// ProcessBlockForStateRoot processes the state for state root computation. It skips proposer signature
// and randao signature verifications.
func ProcessBlockForStateRoot(
	ctx context.Context,
	state *stateTrie.BeaconState,
	signed *ethpb.SignedBeaconBlock,
) (*stateTrie.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.state.ProcessBlock")
	defer span.End()

	state, err := b.ProcessBlockHeaderNoVerify(state, signed.Block)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return nil, errors.Wrap(err, "could not process block header")
	}

	state, err = b.ProcessRandaoNoVerify(state, signed.Block.Body)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return nil, errors.Wrap(err, "could not verify and process randao")
	}

	state, err = b.ProcessEth1DataInBlock(state, signed.Block)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return nil, errors.Wrap(err, "could not process eth1 data")
	}

	state, err = ProcessOperationsNoVerify(ctx, state, signed.Block.Body)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return nil, errors.Wrap(err, "could not process block operation")
	}

	return state, nil
}
