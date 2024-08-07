package electra

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/blocks"
	v "github.com/prysmaticlabs/prysm/v5/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
)

var (
	ProcessBLSToExecutionChanges = blocks.ProcessBLSToExecutionChanges
	ProcessVoluntaryExits        = blocks.ProcessVoluntaryExits
	ProcessAttesterSlashings     = blocks.ProcessAttesterSlashings
	ProcessProposerSlashings     = blocks.ProcessProposerSlashings
	ProcessPayloadAttestations   = blocks.ProcessAttestationsNoVerifySignature
)

// ProcessOperations
//
// Spec definition:
//
//  def process_operations(state: BeaconState, body: BeaconBlockBody) -> None:
//      # [Modified in Electra:EIP6110]
//      # Disable former deposit mechanism once all prior deposits are processed
//      eth1_deposit_index_limit = min(state.eth1_data.deposit_count, state.deposit_requests_start_index)
//      if state.eth1_deposit_index < eth1_deposit_index_limit:
//          assert len(body.deposits) == min(MAX_DEPOSITS, eth1_deposit_index_limit - state.eth1_deposit_index)
//      else:
//          assert len(body.deposits) == 0
//
//      def for_ops(operations: Sequence[Any], fn: Callable[[BeaconState, Any], None]) -> None:
//          for operation in operations:
//              fn(state, operation)
//
//      for_ops(body.proposer_slashings, process_proposer_slashing)
//      for_ops(body.attester_slashings, process_attester_slashing)
//      for_ops(body.attestations, process_attestation)  # [Modified in Electra:EIP7549]
//      for_ops(body.deposits, process_deposit)  # [Modified in Electra:EIP7251]
//      for_ops(body.voluntary_exits, process_voluntary_exit)  # [Modified in Electra:EIP7251]
//      for_ops(body.bls_to_execution_changes, process_bls_to_execution_change)
//		# Removed `process_deposit_request` in EIP-7732
// 		# Removed `process_withdrawal_request` in EIP-7732
// 		# Removed `process_consolidation_request` in EIP-7732
//      for_ops(body.payload_attestations, process_payload_attestation)  # [New in EIP-7732]

func ProcessOperations(
	ctx context.Context,
	st state.BeaconState,
	block interfaces.ReadOnlyBeaconBlock) (state.BeaconState, error) {
	// 6110 validations are in VerifyOperationLengths
	bb := block.Body()
	// Electra extends the altair operations.
	st, err := ProcessProposerSlashings(ctx, st, bb.ProposerSlashings(), v.SlashValidator)
	if err != nil {
		return nil, errors.Wrap(err, "could not process altair proposer slashing")
	}
	st, err = ProcessAttesterSlashings(ctx, st, bb.AttesterSlashings(), v.SlashValidator)
	if err != nil {
		return nil, errors.Wrap(err, "could not process altair attester slashing")
	}
	st, err = ProcessAttestationsNoVerifySignature(ctx, st, block)
	if err != nil {
		return nil, errors.Wrap(err, "could not process altair attestation")
	}
	if _, err := ProcessDeposits(ctx, st, bb.Deposits()); err != nil { // new in electra
		return nil, errors.Wrap(err, "could not process altair deposit")
	}
	st, err = ProcessVoluntaryExits(ctx, st, bb.VoluntaryExits())
	if err != nil {
		return nil, errors.Wrap(err, "could not process voluntary exits")
	}
	st, err = ProcessBLSToExecutionChanges(st, block)
	if err != nil {
		return nil, errors.Wrap(err, "could not process bls-to-execution changes")
	}
	// new in epbs
	st, err = ProcessPayloadAttestations(ctx, st, block)
	if err != nil {
		return nil, errors.Wrap(err, "could not process payload attestations")
	}
	return st, nil
}
