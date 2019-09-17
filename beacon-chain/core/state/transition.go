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
	"github.com/prysmaticlabs/go-ssz"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	e "github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/interop"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
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
	helpers.ClearStartShardCache()
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
			return nil, errors.Wrap(err, "could not process block")
		}
	}

	interop.WriteStateToDisk(state)

	postStateRoot, err := ssz.HashTreeRoot(state)
	if err != nil {
		return nil, errors.Wrap(err, "could not tree hash processed state")
	}
	if !bytes.Equal(postStateRoot[:], block.StateRoot) {
		return state, fmt.Errorf("validate state root failed, wanted: %#x, received: %#x",
			postStateRoot[:], block.StateRoot)
	}

	return state, nil
}

// ExecuteStateTransitionNoVerify defines the procedure for a state transition function.
// This does not validate state root, The use case of such is for state root calculation, the proposer
// should first run state transition on an unsigned block containing a stub for the state root and signature.
// This does not modify state.
//
// WARNING: This method does not validate state root and proposer signature. This is used for proposer to compute
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
func ExecuteStateTransitionNoVerify(
	ctx context.Context,
	state *pb.BeaconState,
	block *ethpb.BeaconBlock,
) (*pb.BeaconState, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	stateCopy := proto.Clone(state).(*pb.BeaconState)
	helpers.ClearStartShardCache()
	b.ClearEth1DataVoteCache()
	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.ExecuteStateTransition")
	defer span.End()
	var err error

	// Execute per slots transition.
	stateCopy, err = ProcessSlots(ctx, stateCopy, block.Slot)
	if err != nil {
		return nil, errors.Wrap(err, "could not process slot")
	}

	// Execute per block transition.
	if block != nil {
		stateCopy, err = processBlockNoVerify(ctx, stateCopy, block)
		if err != nil {
			return nil, errors.Wrap(err, "could not process block")
		}
	}

	return stateCopy, nil
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

	prevStateRoot, err := ssz.HashTreeRoot(state)
	if err != nil {
		return nil, errors.Wrap(err, "could not tree hash prev state root")
	}
	state.StateRoots[state.Slot%params.BeaconConfig().SlotsPerHistoricalRoot] = prevStateRoot[:]

	zeroHash := params.BeaconConfig().ZeroHash
	// Cache latest block header state root.
	if bytes.Equal(state.LatestBlockHeader.StateRoot, zeroHash[:]) {
		state.LatestBlockHeader.StateRoot = prevStateRoot[:]
	}
	prevBlockRoot, err := ssz.SigningRoot(state.LatestBlockHeader)
	if err != nil {
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
	if state.Slot > slot {
		return nil, fmt.Errorf("expected state.slot %d < slot %d", state.Slot, slot)
	}
	for state.Slot < slot {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		state, err := ProcessSlot(ctx, state)
		if err != nil {
			return nil, errors.Wrap(err, "could not process slot")
		}
		if CanProcessEpoch(state) {
			state, err = ProcessEpoch(ctx, state)
			if err != nil {
				return nil, errors.Wrap(err, "could not process epoch")
			}
		}
		state.Slot++
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
		return nil, errors.Wrap(err, "could not process block header")
	}

	state, err = b.ProcessRandao(state, block.Body)
	if err != nil {
		return nil, errors.Wrap(err, "could not verify and process randao")
	}

	state, err = b.ProcessEth1DataInBlock(state, block)
	if err != nil {
		return nil, errors.Wrap(err, "could not process eth1 data")
	}

	state, err = ProcessOperations(ctx, state, block.Body)
	if err != nil {
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
		return nil, errors.Wrap(err, "could not process block header")
	}

	state, err = b.ProcessRandao(state, block.Body)
	if err != nil {
		return nil, errors.Wrap(err, "could not verify and process randao")
	}

	state, err = b.ProcessEth1DataInBlock(state, block)
	if err != nil {
		return nil, errors.Wrap(err, "could not process eth1 data")
	}

	state, err = processOperationsNoVerify(ctx, state, block.Body)
	if err != nil {
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

	// Verify that there are no duplicate transfers
	transferSet := make(map[[32]byte]bool)
	for _, transfer := range body.Transfers {
		h, err := hashutil.HashProto(transfer)
		if err != nil {
			return nil, errors.Wrap(err, "could not hash transfer")
		}
		if transferSet[h] {
			return nil, fmt.Errorf("duplicate transfer: %v", transfer)
		}
		transferSet[h] = true
	}

	state, err := b.ProcessProposerSlashings(state, body)
	if err != nil {
		return nil, errors.Wrap(err, "could not process block proposer slashings")
	}
	state, err = b.ProcessAttesterSlashings(state, body)
	if err != nil {
		return nil, errors.Wrap(err, "could not process block attester slashings")
	}
	state, err = b.ProcessAttestations(state, body)
	if err != nil {
		return nil, errors.Wrap(err, "could not process block attestations")
	}
	state, err = b.ProcessDeposits(state, body)
	if err != nil {
		return nil, errors.Wrap(err, "could not process block validator deposits")
	}
	state, err = b.ProcessVoluntaryExits(state, body)
	if err != nil {
		return nil, errors.Wrap(err, "could not process validator exits")
	}
	state, err = b.ProcessTransfers(state, body)
	if err != nil {
		return nil, errors.Wrap(err, "could not process block transfers")
	}

	return state, nil
}

// processOperationsNoVerify processes the operations in the beacon block and updates beacon state
// with the operations in block. It does not validate the attestation signature
//
// WARNING: This method does not verify attestation signature. This is used for proposer to compute state root
// using a unsigned block.
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

	// Verify that there are no duplicate transfers
	transferSet := make(map[[32]byte]bool)
	for _, transfer := range body.Transfers {
		h, err := hashutil.HashProto(transfer)
		if err != nil {
			return nil, errors.Wrap(err, "could not hash transfer")
		}
		if transferSet[h] {
			return nil, fmt.Errorf("duplicate transfer: %v", transfer)
		}
		transferSet[h] = true
	}

	state, err := b.ProcessProposerSlashings(state, body)
	if err != nil {
		return nil, errors.Wrap(err, "could not process block proposer slashings")
	}
	state, err = b.ProcessAttesterSlashings(state, body)
	if err != nil {
		return nil, errors.Wrap(err, "could not process block attester slashings")
	}
	state, err = b.ProcessAttestationsNoVerify(state, body)
	if err != nil {
		return nil, errors.Wrap(err, "could not process block attestations")
	}
	state, err = b.ProcessDeposits(state, body)
	if err != nil {
		return nil, errors.Wrap(err, "could not process block validator deposits")
	}
	state, err = b.ProcessVoluntaryExits(state, body)
	if err != nil {
		return nil, errors.Wrap(err, "could not process validator exits")
	}
	state, err = b.ProcessTransfers(state, body)
	if err != nil {
		return nil, errors.Wrap(err, "could not process block transfers")
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

	if uint64(len(body.Transfers)) > params.BeaconConfig().MaxTransfers {
		return fmt.Errorf(
			"number of transfers (%d) in block body exceeds allowed threshold of %d",
			len(body.Transfers),
			params.BeaconConfig().MaxTransfers,
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

	state, err = e.ProcessCrosslinks(state)
	if err != nil {
		return nil, errors.Wrap(err, "could not process crosslink")
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
