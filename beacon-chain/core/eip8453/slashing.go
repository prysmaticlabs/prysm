package eip8453

import (
	"context"
	"fmt"
	"slices"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/helpers"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/time"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/validators"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/state"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/pkg/errors"
)

// ProcessAttesterSlashing slashes the validators in both attestations of a finality gadget slashing.
func ProcessAttesterSlashing(ctx context.Context, st state.BeaconState, slashing *ethpb.AttesterSlashingEIP8453) error {
	first, second := slashing.Attestation_1, slashing.Attestation_2
	if !isSlashable(first.Data, second.Data) {
		return errors.New("attestations are not slashable")
	}
	if err := verifyIndexedAttestation(st, first); err != nil {
		return errors.Wrap(err, "invalid first attestation")
	}
	if err := verifyIndexedAttestation(st, second); err != nil {
		return errors.Wrap(err, "invalid second attestation")
	}
	return slashCommon(ctx, st, first.AttestingIndices, second.AttestingIndices)
}

// ProcessLegacyAttesterSlashing slashes the validators in both attestations of a Casper FFG slashing made before EIP-8453.
func ProcessLegacyAttesterSlashing(ctx context.Context, st state.BeaconState, slashing *ethpb.AttesterSlashingGloas) error {
	first, second := slashing.Attestation_1, slashing.Attestation_2
	slashable, err := isSlashableLegacy(first.Data, second.Data)
	if err != nil {
		return err
	}
	if !slashable {
		return errors.New("attestations are not slashable")
	}
	if err := verifyLegacyIndexedAttestation(st, first); err != nil {
		return errors.Wrap(err, "invalid first attestation")
	}
	if err := verifyLegacyIndexedAttestation(st, second); err != nil {
		return errors.Wrap(err, "invalid second attestation")
	}
	return slashCommon(ctx, st, first.AttestingIndices, second.AttestingIndices)
}

func isSlashable(first, second *ethpb.AttestationDataEIP8453) bool {
	return isDoubleTargetVote(first, second) ||
		conflictsWithFinality(first, second.Finality) ||
		conflictsWithFinality(second, first.Finality)
}

func isDoubleTargetVote(first, second *ethpb.AttestationDataEIP8453) bool {
	if first.Timeout || second.Timeout {
		return false
	}
	return first.Target.Height == second.Target.Height && !rootsEqual(first.Target.Root, second.Target.Root)
}

func conflictsWithFinality(data *ethpb.AttestationDataEIP8453, finality *ethpb.HeightCheckpoint) bool {
	if isEmptyCheckpoint(finality) || data.Target.Height != finality.Height {
		return false
	}
	if data.Timeout {
		return true
	}
	return !rootsEqual(data.Target.Root, finality.Root)
}

func isSlashableLegacy(first, second *ethpb.AttestationData) (bool, error) {
	firstRoot, err := first.HashTreeRoot()
	if err != nil {
		return false, err
	}
	secondRoot, err := second.HashTreeRoot()
	if err != nil {
		return false, err
	}
	doubleVote := firstRoot != secondRoot && first.Target.Epoch == second.Target.Epoch
	surroundVote := first.Source.Epoch < second.Source.Epoch && second.Target.Epoch < first.Target.Epoch
	return doubleVote || surroundVote, nil
}

func verifyLegacyIndexedAttestation(st state.ReadOnlyBeaconState, att *ethpb.IndexedAttestationGloas) error {
	indices := att.AttestingIndices
	if len(indices) == 0 {
		return errors.New("no attesting indices")
	}
	cfg := params.BeaconConfig()
	if uint64(len(indices)) > cfg.MaxValidatorsPerCommittee*cfg.MaxCommitteesPerSlot {
		return fmt.Errorf("%d attesting indices, over the limit", len(indices))
	}
	for i := 1; i < len(indices); i++ {
		if indices[i-1] >= indices[i] {
			return errors.New("attesting indices are not sorted and unique")
		}
	}
	return verifyAggregate(st, indices, cfg.DomainBeaconAttester, att.Data.Target.Epoch, att.Data, att.Signature)
}

func slashCommon(ctx context.Context, st state.BeaconState, first, second []uint64) error {
	epoch := time.CurrentEpoch(st)
	exitInfo := validators.ExitInformation(st)
	slashedAny := false
	for _, index := range first {
		if _, ok := slices.BinarySearch(second, index); !ok {
			continue
		}
		val, err := st.ValidatorAtIndexReadOnly(primitives.ValidatorIndex(index))
		if err != nil {
			return err
		}
		if !helpers.IsSlashableValidatorUsingTrie(val, epoch) {
			continue
		}
		if err := slashValidator(ctx, st, primitives.ValidatorIndex(index), exitInfo); err != nil {
			return err
		}
		slashedAny = true
	}
	if !slashedAny {
		return errors.New("no validator was slashed")
	}
	return nil
}

// Gloas beta.5 moved the builder payment removal into slash_validator, Prysm's Gloas still removes it per proposer slashing.
func slashValidator(ctx context.Context, st state.BeaconState, index primitives.ValidatorIndex, exitInfo *validators.ExitInfo) error {
	if _, err := validators.SlashValidator(ctx, st, index, exitInfo); err != nil {
		return err
	}
	payments, err := st.BuilderPendingPayments()
	if err != nil {
		return err
	}
	for i, payment := range payments {
		if payment.ProposerIndex != index {
			continue
		}
		if err := st.ClearBuilderPendingPayment(primitives.Slot(i)); err != nil {
			return err
		}
	}
	return nil
}
