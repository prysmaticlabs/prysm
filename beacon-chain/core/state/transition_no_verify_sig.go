package state

import (
	"bytes"
	"context"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// ExecuteStateTransitionNoVerifyAnySig defines the procedure for a state transition function.
// This does not validate any BLS signatures of attestations, block proposer signature, randao signature,
// it is used for performing a state transition as quickly as possible. This function also returns a signature
// set of all signatures not verified, so that they can be stored and verified later.
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
func ExecuteStateTransitionNoVerifyAnySig(
	ctx context.Context,
	state iface.BeaconState,
	signed *ethpb.SignedBeaconBlock,
) (*bls.SignatureSet, iface.BeaconState, error) {
	if ctx.Err() != nil {
		return nil, nil, ctx.Err()
	}
	if signed == nil || signed.Block == nil {
		return nil, nil, errors.New("nil block")
	}

	ctx, span := trace.StartSpan(ctx, "core.state.ExecuteStateTransitionNoVerifyAttSigs")
	defer span.End()
	var err error

	if featureconfig.Get().EnableNextSlotStateCache {
		state, err = ProcessSlotsUsingNextSlotCache(ctx, state, signed.Block.ParentRoot, signed.Block.Slot)
		if err != nil {
			return nil, nil, errors.Wrap(err, "could not process slots")
		}
	} else {
		state, err = ProcessSlots(ctx, state, signed.Block.Slot)
		if err != nil {
			return nil, nil, errors.Wrap(err, "could not process slot")
		}
	}

	// Execute per block transition.
	set, state, err := ProcessBlockNoVerifyAnySig(ctx, state, signed)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not process block")
	}

	// State root validation.
	postStateRoot, err := state.HashTreeRoot(ctx)
	if err != nil {
		return nil, nil, err
	}
	if !bytes.Equal(postStateRoot[:], signed.Block.StateRoot) {
		return nil, nil, fmt.Errorf("could not validate state root, wanted: %#x, received: %#x",
			postStateRoot[:], signed.Block.StateRoot)
	}

	return set, state, nil
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
	state iface.BeaconState,
	signed *ethpb.SignedBeaconBlock,
) ([32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "core.state.CalculateStateRoot")
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
	var err error
	if featureconfig.Get().EnableNextSlotStateCache {
		state, err = ProcessSlotsUsingNextSlotCache(ctx, state, signed.Block.ParentRoot, signed.Block.Slot)
		if err != nil {
			return [32]byte{}, errors.Wrap(err, "could not process slots")
		}
	} else {
		state, err = ProcessSlots(ctx, state, signed.Block.Slot)
		if err != nil {
			return [32]byte{}, errors.Wrap(err, "could not process slot")
		}
	}

	// Execute per block transition.
	state, err = ProcessBlockForStateRoot(ctx, state, signed)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not process block")
	}

	return state.HashTreeRoot(ctx)
}

// ProcessBlockNoVerifyAnySig creates a new, modified beacon state by applying block operation
// transformations as defined in the Ethereum Serenity specification. It does not validate
// any block signature except for deposit and slashing signatures. It also returns the relevant
// signature set from all the respective methods.
//
// Spec pseudocode definition:
//
//  def process_block(state: BeaconState, block: BeaconBlock) -> None:
//    process_block_header(state, block)
//    process_randao(state, block.body)
//    process_eth1_data(state, block.body)
//    process_operations(state, block.body)
func ProcessBlockNoVerifyAnySig(
	ctx context.Context,
	state iface.BeaconState,
	signed *ethpb.SignedBeaconBlock,
) (*bls.SignatureSet, iface.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "core.state.ProcessBlockNoVerifyAnySig")
	defer span.End()

	state, err := b.ProcessBlockHeaderNoVerify(state, signed.Block)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return nil, nil, errors.Wrap(err, "could not process block header")
	}
	bSet, err := b.BlockSignatureSet(state, signed)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return nil, nil, errors.Wrap(err, "could not retrieve block signature set")
	}
	rSet, err := b.RandaoSignatureSet(state, signed.Block.Body)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return nil, nil, errors.Wrap(err, "could not retrieve randao signature set")
	}
	state, err = b.ProcessRandaoNoVerify(state, signed.Block.Body)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return nil, nil, errors.Wrap(err, "could not verify and process randao")
	}

	state, err = b.ProcessEth1DataInBlock(ctx, state, signed)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return nil, nil, errors.Wrap(err, "could not process eth1 data")
	}

	state, err = ProcessOperationsNoVerifyAttsSigs(ctx, state, signed)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return nil, nil, errors.Wrap(err, "could not process block operation")
	}
	aSet, err := b.AttestationSignatureSet(ctx, state, signed.Block.Body.Attestations)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not retrieve attestation signature set")
	}

	// Merge beacon block, randao and attestations signatures into a set.
	set := bls.NewSet()
	set.Join(bSet).Join(rSet).Join(aSet)

	return set, state, nil
}

// ProcessOperationsNoVerifyAttsSigs processes the operations in the beacon block and updates beacon state
// with the operations in block. It does not verify attestation signatures.
//
// WARNING: This method does not verify attestation signatures.
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
func ProcessOperationsNoVerifyAttsSigs(
	ctx context.Context,
	state iface.BeaconState,
	signedBeaconBlock *ethpb.SignedBeaconBlock) (iface.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "core.state.ProcessOperationsNoVerifyAttsSigs")
	defer span.End()

	if _, err := VerifyOperationLengths(ctx, state, signedBeaconBlock); err != nil {
		return nil, errors.Wrap(err, "could not verify operation lengths")
	}

	state, err := b.ProcessProposerSlashings(ctx, state, signedBeaconBlock)
	if err != nil {
		return nil, errors.Wrap(err, "could not process block proposer slashings")
	}
	state, err = b.ProcessAttesterSlashings(ctx, state, signedBeaconBlock)
	if err != nil {
		return nil, errors.Wrap(err, "could not process block attester slashings")
	}
	state, err = b.ProcessAttestationsNoVerifySignature(ctx, state, signedBeaconBlock)
	if err != nil {
		return nil, errors.Wrap(err, "could not process block attestations")
	}
	state, err = b.ProcessDeposits(ctx, state, signedBeaconBlock.Block.Body.Deposits)
	if err != nil {
		return nil, errors.Wrap(err, "could not process block validator deposits")
	}
	state, err = b.ProcessVoluntaryExits(ctx, state, signedBeaconBlock)
	if err != nil {
		return nil, errors.Wrap(err, "could not process validator exits")
	}

	return state, nil
}

// ProcessBlockForStateRoot processes the state for state root computation. It skips proposer signature
// and randao signature verifications.
func ProcessBlockForStateRoot(
	ctx context.Context,
	state iface.BeaconState,
	signed *ethpb.SignedBeaconBlock,
) (iface.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "core.state.ProcessBlockForStateRoot")
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

	state, err = b.ProcessEth1DataInBlock(ctx, state, signed)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return nil, errors.Wrap(err, "could not process eth1 data")
	}

	state, err = ProcessOperationsNoVerifyAttsSigs(ctx, state, signed)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return nil, errors.Wrap(err, "could not process block operation")
	}

	return state, nil
}
