package altair

import (
	"context"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	e "github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch/precompute"
	s "github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

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
	signed *ethpb.SignedBeaconBlockAltair,
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
//
//    def for_ops(operations: Sequence[Any], fn: Callable[[BeaconState, Any], None]) -> None:
//        for operation in operations:
//            fn(state, operation)
//
//    for_ops(body.proposer_slashings, process_proposer_slashing)
//    for_ops(body.attester_slashings, process_attester_slashing)
//    for_ops(body.attestations, process_attestation)
//    for_ops(body.deposits, process_deposit)
//    for_ops(body.voluntary_exits, process_voluntary_exit)
func ProcessOperationsNoVerifyAttsSigs(
	ctx context.Context,
	state iface.BeaconState,
	signedBeaconBlock *ethpb.SignedBeaconBlockAltair) (iface.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "core.state.ProcessOperationsNoVerifyAttsSigs")
	defer span.End()

	if _, err := s.VerifyOperationLengths(ctx, state, signedBeaconBlock); err != nil {
		return nil, errors.Wrap(err, "could not verify operation lengths")
	}

	// Modified in Altair.
	state, err := b.ProcessProposerSlashings(ctx, state, signedBeaconBlock, SlashValidator)
	if err != nil {
		return nil, errors.Wrap(err, "could not process block proposer slashings")
	}
	// Modified in Altair.
	state, err = b.ProcessAttesterSlashings(ctx, state, signedBeaconBlock, SlashValidator)
	if err != nil {
		return nil, errors.Wrap(err, "could not process block attester slashings")
	}
	// Modified in Altair.
	state, err = ProcessAttestationsNoVerifySignature(ctx, state, signedBeaconBlock)
	if err != nil {
		return nil, errors.Wrap(err, "could not process block attestations")
	}
	// Modified in Altair.
	state, err = ProcessDeposits(ctx, state, signedBeaconBlock.Block.Body.Deposits)
	if err != nil {
		return nil, errors.Wrap(err, "could not process block validator deposits")
	}
	state, err = b.ProcessVoluntaryExits(ctx, state, signedBeaconBlock)
	if err != nil {
		return nil, errors.Wrap(err, "could not process validator exits")
	}
	return state, nil
}

// ProcessEpoch describes the per epoch operations that are performed on the beacon state.
// It's optimized by pre computing validator attested info and epoch total/attested balances upfront.
//
// Spec code:
// def process_epoch(state: BeaconState) -> None:
//    process_justification_and_finalization(state)  # [Modified in Altair]
//    process_inactivity_updates(state)  # [New in Altair]
//    process_rewards_and_penalties(state)  # [Modified in Altair]
//    process_registry_updates(state)
//    process_slashings(state)  # [Modified in Altair]
//    process_eth1_data_reset(state)
//    process_effective_balance_updates(state)
//    process_slashings_reset(state)
//    process_randao_mixes_reset(state)
//    process_historical_roots_update(state)
//    process_participation_flag_updates(state)  # [New in Altair]
//    process_sync_committee_updates(state)  # [New in Altair]
func ProcessEpoch(ctx context.Context, state iface.BeaconStateAltair) (iface.BeaconStateAltair, error) {
	ctx, span := trace.StartSpan(ctx, "altair.ProcessEpoch")
	defer span.End()

	if state == nil {
		return nil, errors.New("nil state")
	}
	vp, bp, err := InitializeEpochValidators(ctx, state)
	if err != nil {
		return nil, err
	}

	// New in Altair.
	vp, bp, err = ProcessEpochParticipation(ctx, state, bp, vp)
	if err != nil {
		return nil, err
	}

	state, err = precompute.ProcessJustificationAndFinalizationPreCompute(state, bp)
	if err != nil {
		return nil, errors.Wrap(err, "could not process justification")
	}

	// New in Altair.
	// process_inactivity_updates is embedded in the below.
	state, err = ProcessRewardsAndPenaltiesPrecompute(state, bp, vp)
	if err != nil {
		return nil, errors.Wrap(err, "could not process rewards and penalties")
	}

	state, err = e.ProcessRegistryUpdates(state)
	if err != nil {
		return nil, errors.Wrap(err, "could not process registry updates")
	}

	// Modified in Altair.
	state, err = ProcessSlashings(state)
	if err != nil {
		return nil, err
	}

	state, err = e.ProcessEth1DataReset(state)
	if err != nil {
		return nil, err
	}
	state, err = e.ProcessEffectiveBalanceUpdates(state)
	if err != nil {
		return nil, err
	}
	state, err = e.ProcessSlashingsReset(state)
	if err != nil {
		return nil, err
	}
	state, err = e.ProcessRandaoMixesReset(state)
	if err != nil {
		return nil, err
	}
	state, err = e.ProcessHistoricalRootsUpdate(state)
	if err != nil {
		return nil, err
	}

	// New in Altair.
	state, err = ProcessParticipationFlagUpdates(state)
	if err != nil {
		return nil, err
	}

	// New in Altair.
	state, err = ProcessSyncCommitteeUpdates(state)
	if err != nil {
		return nil, err
	}

	return state, nil
}
