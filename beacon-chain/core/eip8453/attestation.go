package eip8453

import (
	"bytes"
	"context"
	"fmt"
	"slices"

	"github.com/OffchainLabs/go-bitfield"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/helpers"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/time"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/state"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/pkg/errors"
)

// ProcessAttestation records the target, progress and finality votes of an aggregate and rewards the proposer.
func ProcessAttestation(ctx context.Context, st state.BeaconState, att *ethpb.AttestationEIP8453) error {
	data := att.Data
	if !isWellFormed(data) {
		return errors.New("attestation data is not well formed")
	}
	target, err := st.TargetCheckpoint()
	if err != nil {
		return err
	}
	pendings, err := st.PendingFinalities()
	if err != nil {
		return err
	}
	if !isValidData(data, target, pendings) {
		return errors.New("attestation votes for neither the current target nor a pending justification")
	}
	if att.Epoch > time.CurrentEpoch(st) {
		return fmt.Errorf("attestation epoch %d is in the future", att.Epoch)
	}
	finalized, err := finalizedEpoch(st)
	if err != nil {
		return err
	}
	if att.Epoch < finalized {
		return fmt.Errorf("attestation epoch %d is before the finalized epoch %d", att.Epoch, finalized)
	}
	indices, err := aggregateParticipants(st, att)
	if err != nil {
		return err
	}
	indexed := &ethpb.IndexedAttestationEIP8453{AttestingIndices: indices, Data: data, Signature: att.Signature, Epoch: att.Epoch}
	if err := verifyIndexedAttestation(st, indexed); err != nil {
		return errors.Wrap(err, "invalid indexed attestation")
	}

	progress, err := st.ProgressParticipation()
	if err != nil {
		return err
	}
	targetVotes, err := st.TargetParticipationEIP8453()
	if err != nil {
		return err
	}
	round, err := st.RoundParticipation()
	if err != nil {
		return err
	}
	progress, targetVotes, round = slices.Clone(progress), slices.Clone(targetVotes), slices.Clone(round)
	pendings = slices.Clone(pendings)
	for i, pending := range pendings {
		if checkpointsEqual(pending.Checkpoint, data.Finality) {
			pendings[i] = pending.Copy()
		}
	}
	onTarget := checkpointsEqual(data.Target, target)
	total, err := helpers.TotalActiveBalance(ctx, st)
	if err != nil {
		return err
	}
	var numerator uint64
	for _, index := range indices {
		if index >= uint64(len(round)) {
			return fmt.Errorf("validator %d has no round participation", index)
		}
		eb, err := st.EffectiveBalanceAtIndex(primitives.ValidatorIndex(index))
		if err != nil {
			return err
		}
		baseReward := roundBaseReward(eb, total)
		for _, pending := range pendings {
			if !checkpointsEqual(pending.Checkpoint, data.Finality) {
				continue
			}
			if index >= uint64(len(pending.Participation)) {
				return fmt.Errorf("validator %d has no finality participation", index)
			}
			if pending.Participation[index] == 0 {
				pending.Participation[index] = 1
				round[index] |= 1 << finalityFlagIndex
				numerator += baseReward * finalityWeight
			}
		}
		if !onTarget {
			continue
		}
		if index >= uint64(len(progress)) || index >= uint64(len(targetVotes)) {
			return fmt.Errorf("validator %d has no height participation", index)
		}
		if progress[index] == 0 {
			progress[index] = 1
			round[index] |= 1 << progressFlagIndex
			numerator += baseReward * progressWeight
		}
		if !data.Timeout {
			targetVotes[index] = 1
		}
	}
	if err := st.SetPendingFinalities(pendings); err != nil {
		return err
	}
	if err := st.SetProgressParticipation(progress); err != nil {
		return err
	}
	if err := st.SetTargetParticipationEIP8453(targetVotes); err != nil {
		return err
	}
	if err := st.SetRoundParticipation(round); err != nil {
		return err
	}

	cfg := params.BeaconConfig()
	divisor := (cfg.WeightDenominator - cfg.ProposerWeight) * cfg.WeightDenominator / cfg.ProposerWeight
	proposer, err := helpers.BeaconProposerIndex(ctx, st)
	if err != nil {
		return err
	}
	return helpers.IncreaseBalance(st, proposer, numerator/divisor)
}

func isWellFormed(data *ethpb.AttestationDataEIP8453) bool {
	if data.Target.Height == 0 || isZeroRoot(data.Target.Root) {
		return false
	}
	if isEmptyCheckpoint(data.Finality) {
		return true
	}
	return !isZeroRoot(data.Finality.Root) && data.Finality.Height < data.Target.Height
}

func isValidData(data *ethpb.AttestationDataEIP8453, target *ethpb.HeightCheckpoint, pendings []*ethpb.PendingFinalityEIP8453) bool {
	if checkpointsEqual(data.Target, target) {
		return true
	}
	pending := slices.ContainsFunc(pendings, func(p *ethpb.PendingFinalityEIP8453) bool {
		return checkpointsEqual(p.Checkpoint, data.Finality)
	})
	return pending && data.Finality.Height < data.Target.Height && data.Target.Height < target.Height
}

func aggregateParticipants(st state.ReadOnlyBeaconState, att *ethpb.AttestationEIP8453) ([]uint64, error) {
	active := activeIndices(st, att.Epoch)
	var committees [][]primitives.ValidatorIndex
	var size uint64
	for _, index := range committeeIndices(att.CommitteeBits) {
		committee := attestationCommittee(active, att.Epoch, index)
		committees = append(committees, committee)
		size += uint64(len(committee))
	}
	if size > params.BeaconConfig().MaxValidatorsPerAggregate {
		return nil, fmt.Errorf("aggregate covers %d validators, over the limit", size)
	}
	bits := bitfield.Bitlist(att.AggregationBits)
	if bits.Len() != size {
		return nil, fmt.Errorf("aggregation bits length %d does not match committees size %d", bits.Len(), size)
	}
	seen := make(map[uint64]bool)
	var offset uint64
	for _, committee := range committees {
		for i, index := range committee {
			if bits.BitAt(offset + uint64(i)) {
				seen[uint64(index)] = true
			}
		}
		offset += uint64(len(committee))
	}
	indices := make([]uint64, 0, len(seen))
	for index := range seen {
		indices = append(indices, index)
	}
	slices.Sort(indices)
	return indices, nil
}

func verifyIndexedAttestation(st state.ReadOnlyBeaconState, att *ethpb.IndexedAttestationEIP8453) error {
	indices := att.AttestingIndices
	if len(indices) == 0 {
		return errors.New("no attesting indices")
	}
	if uint64(len(indices)) > params.BeaconConfig().MaxValidatorsPerAggregate {
		return fmt.Errorf("%d attesting indices, over the limit", len(indices))
	}
	for i := 1; i < len(indices); i++ {
		if indices[i-1] >= indices[i] {
			return errors.New("attesting indices are not sorted and unique")
		}
	}
	return verifyAggregate(st, indices, params.BeaconConfig().DomainFinalityAttester, att.Epoch, att.Data, att.Signature)
}

func checkpointsEqual(a, b *ethpb.HeightCheckpoint) bool {
	return a.GetHeight() == b.GetHeight() && rootsEqual(a.GetRoot(), b.GetRoot())
}

func rootsEqual(a, b []byte) bool {
	return bytes.Equal(rootOrZero(a), rootOrZero(b))
}

func isEmptyCheckpoint(c *ethpb.HeightCheckpoint) bool {
	return c.GetHeight() == 0 && isZeroRoot(c.GetRoot())
}

func isZeroRoot(root []byte) bool {
	return bytes.Equal(rootOrZero(root), make([]byte, 32))
}

func rootOrZero(root []byte) []byte {
	if len(root) == 0 {
		return make([]byte, 32)
	}
	return root
}
