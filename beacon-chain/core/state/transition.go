// Package state implements the whole state transition
// function which consists of per slot, per-epoch transitions.
// It also bootstraps the genesis beacon state for slot 0.
package state

import (
	"bytes"
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	e "github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch/precompute"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/interop"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/stateutil"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// ExecuteStateTransition defines the procedure for a state transition function.
//
// Spec pseudocode definition:
//  def state_transition(state: BeaconState, block: BeaconBlock, validate_state_root: bool=False) -> BeaconState:
//    # Process slots (including those with no blocks) since block
//    process_slots(state, block.slot)
//    # Process block
//    process_block(state, block)
//    # Validate state root (`validate_state_root == True` in production)
//    if validate_state_root:
//        assert block.state_root == hash_tree_root(state)
//    # Return post-state
//    return state
func ExecuteStateTransition(
	ctx context.Context,
	state *pb.BeaconState,
	block *ethpb.BeaconBlock,
) (*pb.BeaconState, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	b.ClearEth1DataVoteCache()
	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.ExecuteStateTransition")
	defer span.End()
	var err error
	// Execute per slots transition.
	state, err = ProcessSlots(ctx, state, block.Slot)
	if err != nil {
		return nil, errors.Wrap(err, "could not process slot")
	}

	// Execute per block transition.
	if block != nil {
		state, err = ProcessBlock(ctx, state, block)
		if err != nil {
			return nil, errors.Wrapf(err, "could not process block in slot %d", block.Slot)
		}
	}

	interop.WriteBlockToDisk(block, false)
	interop.WriteStateToDisk(state)

	var postStateRoot [32]byte
	if featureconfig.Get().EnableCustomStateSSZ {
		postStateRoot, err = stateutil.HashTreeRootState(state)
		if err != nil {
			return nil, errors.Wrap(err, "could not tree hash processed state")
		}
	} else {
		postStateRoot, err = ssz.HashTreeRoot(state)
		if err != nil {
			return nil, errors.Wrap(err, "could not tree hash processed state")
		}
	}
	if !bytes.Equal(postStateRoot[:], block.StateRoot) {
		return state, fmt.Errorf("validate state root failed, wanted: %#x, received: %#x",
			postStateRoot[:], block.StateRoot)
	}

	return state, nil
}

// ExecuteStateTransitionNoVerify defines the procedure for a state transition function.
// This does not validate any BLS signatures in a block, it is used for performing a state transition as quickly
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
func ExecuteStateTransitionNoVerify(
	ctx context.Context,
	state *pb.BeaconState,
	block *ethpb.BeaconBlock,
) (*pb.BeaconState, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	b.ClearEth1DataVoteCache()
	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.ExecuteStateTransitionNoVerify")
	defer span.End()
	var err error

	// Execute per slots transition.
	state, err = ProcessSlots(ctx, state, block.Slot)
	if err != nil {
		return nil, errors.Wrap(err, "could not process slot")
	}

	// Execute per block transition.
	if block != nil {
		state, err = processBlockNoVerify(ctx, state, block)
		if err != nil {
			return nil, errors.Wrap(err, "could not process block")
		}
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
	state *pb.BeaconState,
	block *ethpb.BeaconBlock,
) ([32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.CalculateStateRoot")
	defer span.End()
	if ctx.Err() != nil {
		traceutil.AnnotateError(span, ctx.Err())
		return [32]byte{}, ctx.Err()
	}

	stateCopy := proto.Clone(state).(*pb.BeaconState)
	b.ClearEth1DataVoteCache()

	var err error
	// Execute per slots transition.
	stateCopy, err = ProcessSlots(ctx, stateCopy, block.Slot)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not process slot")
	}

	// Execute per block transition.
	if block != nil {
		stateCopy, err = processBlockNoVerify(ctx, stateCopy, block)
		if err != nil {
			return [32]byte{}, errors.Wrap(err, "could not process block")
		}
	}

	if featureconfig.Get().EnableCustomStateSSZ {
		return stateutil.HashTreeRootState(stateCopy)
	}
	return ssz.HashTreeRoot(stateCopy)
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
//    previous_block_root = signing_root(state.latest_block_header)
//    state.block_roots[state.slot % SLOTS_PER_HISTORICAL_ROOT] = previous_block_root
func ProcessSlot(ctx context.Context, state *pb.BeaconState) (*pb.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.state.ProcessSlot")
	defer span.End()
	span.AddAttributes(trace.Int64Attribute("slot", int64(state.Slot)))

	var prevStateRoot [32]byte
	var err error
	if featureconfig.Get().EnableCustomStateSSZ {
		prevStateRoot, err = stateutil.HashTreeRootState(state)
		if err != nil {
			traceutil.AnnotateError(span, err)
			return nil, errors.Wrap(err, "could not tree hash prev state root")
		}
		if _, err := ssz.HashTreeRoot(state); err != nil {
			return nil, errors.Wrap(err, "could not tree hash processed state")
		}
	} else {
		prevStateRoot, err = ssz.HashTreeRoot(state)
		if err != nil {
			traceutil.AnnotateError(span, err)
			return nil, errors.Wrap(err, "could not tree hash prev state root")
		}
	}
	state.StateRoots[state.Slot%params.BeaconConfig().SlotsPerHistoricalRoot] = prevStateRoot[:]

	zeroHash := params.BeaconConfig().ZeroHash
	// Cache latest block header state root.
	if bytes.Equal(state.LatestBlockHeader.StateRoot, zeroHash[:]) {
		state.LatestBlockHeader.StateRoot = prevStateRoot[:]
	}
	prevBlockRoot, err := ssz.SigningRoot(state.LatestBlockHeader)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return nil, errors.Wrap(err, "could not determine prev block root")
	}
	// Cache the block root.
	state.BlockRoots[state.Slot%params.BeaconConfig().SlotsPerHistoricalRoot] = prevBlockRoot[:]
	return state, nil
}

// ProcessSlots process through skip skips and apply epoch transition when it's needed
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
func ProcessSlots(ctx context.Context, state *pb.BeaconState, slot uint64) (*pb.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.ProcessSlots")
	defer span.End()
	span.AddAttributes(trace.Int64Attribute("slots", int64(slot)-int64(state.Slot)))
	if state.Slot > slot {
		err := fmt.Errorf("expected state.slot %d < slot %d", state.Slot, slot)
		traceutil.AnnotateError(span, err)
		return nil, err
	}
	highestSlot := state.Slot
	var root [32]byte
	var writeToCache bool
	var err error

	if featureconfig.Get().EnableSkipSlotsCache {
		// Restart from cached value, if one exists.
		if featureconfig.Get().EnableCustomStateSSZ {
			root, err = stateutil.HashTreeRootState(state)
			if err != nil {
				return nil, errors.Wrap(err, "could not HashTreeRoot(state)")
			}
		} else {
			root, err = ssz.HashTreeRoot(state)
			if err != nil {
				return nil, errors.Wrap(err, "could not HashTreeRoot(state)")
			}
		}
		cached, ok := skipSlotCache.Get(root)
		// if cache key does not exist, we write it to the cache.
		writeToCache = !ok
		if ok {
			// do not write to cache if state with higher slot exists.
			writeToCache = cached.(*pb.BeaconState).Slot <= slot
			if cached.(*pb.BeaconState).Slot <= slot {
				state = proto.Clone(cached.(*pb.BeaconState)).(*pb.BeaconState)
				highestSlot = state.Slot
				skipSlotCacheHit.Inc()
			} else {
				skipSlotCacheMiss.Inc()
			}
		}
	}

	for state.Slot < slot {
		if ctx.Err() != nil {
			traceutil.AnnotateError(span, ctx.Err())
			if featureconfig.Get().EnableSkipSlotsCache {
				// Cache last best value.
				if highestSlot < state.Slot && writeToCache {
					skipSlotCache.Add(root, proto.Clone(state).(*pb.BeaconState))
				}
			}
			return nil, ctx.Err()
		}
		state, err := ProcessSlot(ctx, state)
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
		state.Slot++
	}

	if featureconfig.Get().EnableSkipSlotsCache {
		// Clone result state so that caches are not mutated.
		if highestSlot < state.Slot && writeToCache {
			skipSlotCache.Add(root, proto.Clone(state).(*pb.BeaconState))
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
	state *pb.BeaconState,
	block *ethpb.BeaconBlock,
) (*pb.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.state.ProcessBlock")
	defer span.End()

	state, err := b.ProcessBlockHeader(state, block)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return nil, errors.Wrap(err, "could not process block header")
	}

	state, err = b.ProcessRandao(state, block.Body)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return nil, errors.Wrap(err, "could not verify and process randao")
	}

	state, err = b.ProcessEth1DataInBlock(state, block)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return nil, errors.Wrap(err, "could not process eth1 data")
	}

	state, err = ProcessOperations(ctx, state, block.Body)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return nil, errors.Wrap(err, "could not process block operation")
	}

	return state, nil
}

// processBlockNoVerify creates a new, modified beacon state by applying block operation
// transformations as defined in the Ethereum Serenity specification. It does not validate
// block signature.
//
//
// WARNING: This method does not verify proposer signature. This is used for proposer to compute state root
// using a unsigned block.
//
// Spec pseudocode definition:
//
//  def process_block(state: BeaconState, block: BeaconBlock) -> None:
//    process_block_header(state, block)
//    process_randao(state, block.body)
//    process_eth1_data(state, block.body)
//    process_operations(state, block.body)
func processBlockNoVerify(
	ctx context.Context,
	state *pb.BeaconState,
	block *ethpb.BeaconBlock,
) (*pb.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.state.ProcessBlock")
	defer span.End()

	state, err := b.ProcessBlockHeaderNoVerify(state, block)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return nil, errors.Wrap(err, "could not process block header")
	}

	state, err = b.ProcessRandaoNoVerify(state, block.Body)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return nil, errors.Wrap(err, "could not verify and process randao")
	}

	state, err = b.ProcessEth1DataInBlock(state, block)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return nil, errors.Wrap(err, "could not process eth1 data")
	}

	state, err = processOperationsNoVerify(ctx, state, block.Body)
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
	state *pb.BeaconState,
	body *ethpb.BeaconBlockBody) (*pb.BeaconState, error) {
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
	state, err = b.ProcessAttestations(ctx, state, body)
	if err != nil {
		return nil, errors.Wrap(err, "could not process block attestations")
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

// processOperationsNoVerify processes the operations in the beacon block and updates beacon state
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
func processOperationsNoVerify(
	ctx context.Context,
	state *pb.BeaconState,
	body *ethpb.BeaconBlockBody) (*pb.BeaconState, error) {
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

func verifyOperationLengths(state *pb.BeaconState, body *ethpb.BeaconBlockBody) error {
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

	if state.Eth1DepositIndex > state.Eth1Data.DepositCount {
		return fmt.Errorf("expected state.deposit_index %d <= eth1data.deposit_count %d", state.Eth1DepositIndex, state.Eth1Data.DepositCount)
	}
	maxDeposits := mathutil.Min(params.BeaconConfig().MaxDeposits, state.Eth1Data.DepositCount-state.Eth1DepositIndex)
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
func CanProcessEpoch(state *pb.BeaconState) bool {
	return (state.Slot+1)%params.BeaconConfig().SlotsPerEpoch == 0
}

// ProcessEpoch describes the per epoch operations that are performed on the
// beacon state. It focuses on the validator registry, adjusting balances, and finalizing slots.
//
// Spec pseudocode definition:
//
//  def process_epoch(state: BeaconState) -> None:
//    process_justification_and_finalization(state)
//    process_crosslinks(state)
//    process_rewards_and_penalties(state)
//    process_registry_updates(state)
//    # @process_reveal_deadlines
//    # @process_challenge_deadlines
//    process_slashings(state)
//    process_final_updates(state)
//    # @after_process_final_updates
func ProcessEpoch(ctx context.Context, state *pb.BeaconState) (*pb.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.state.ProcessEpoch")
	defer span.End()
	span.AddAttributes(trace.Int64Attribute("epoch", int64(helpers.SlotToEpoch(state.Slot))))

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
		return nil, errors.Wrap(err, "could not get attesting balance prev epoch")
	}
	currentEpochAttestedBalance, err := e.AttestingBalance(state, currentEpochAtts.Target)
	if err != nil {
		return nil, errors.Wrap(err, "could not get attesting balance current epoch")
	}

	state, err = e.ProcessJustificationAndFinalization(state, prevEpochAttestedBalance, currentEpochAttestedBalance)
	if err != nil {
		return nil, errors.Wrap(err, "could not process justification")
	}

	state, err = e.ProcessRewardsAndPenalties(state)
	if err != nil {
		return nil, errors.Wrap(err, "could not process rewards and penalties")
	}

	state, err = e.ProcessRegistryUpdates(state)
	if err != nil {
		return nil, errors.Wrap(err, "could not process registry updates")
	}

	state, err = e.ProcessSlashings(state)
	if err != nil {
		return nil, errors.Wrap(err, "could not process slashings")
	}

	state, err = e.ProcessFinalUpdates(state)
	if err != nil {
		return nil, errors.Wrap(err, "could not process final updates")
	}
	return state, nil
}

// ProcessEpochPrecompute describes the per epoch operations that are performed on the beacon state.
// It's optimized by pre computing validator attested info and epoch total/attested balances upfront.
func ProcessEpochPrecompute(ctx context.Context, state *pb.BeaconState) (*pb.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.state.ProcessEpoch")
	defer span.End()
	span.AddAttributes(trace.Int64Attribute("epoch", int64(helpers.SlotToEpoch(state.Slot))))

	vp, bp := precompute.New(ctx, state)
	vp, bp, err := precompute.ProcessAttestations(ctx, state, vp, bp)
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

	state = precompute.ProcessSlashingsPrecompute(state, bp)

	state, err = e.ProcessFinalUpdates(state)
	if err != nil {
		return nil, errors.Wrap(err, "could not process final updates")
	}
	return state, nil
}
