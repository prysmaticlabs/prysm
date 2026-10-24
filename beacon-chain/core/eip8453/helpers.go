package eip8453

import (
	"encoding/binary"
	"fmt"

	"github.com/OffchainLabs/methodical-ssz/ssz"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/helpers"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/signing"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/time"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/state"
	fieldparams "github.com/OffchainLabs/prysm/v7/config/fieldparams"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/crypto/bls"
	"github.com/OffchainLabs/prysm/v7/crypto/hash"
	"github.com/OffchainLabs/prysm/v7/encoding/bytesutil"
	"github.com/OffchainLabs/prysm/v7/math"
	"github.com/OffchainLabs/prysm/v7/time/slots"
	"github.com/pkg/errors"
)

const (
	progressFlagIndex = 0
	finalityFlagIndex = 1
	progressWeight    = 40
	finalityWeight    = 14
)

var flagWeights = [...]uint64{progressWeight, finalityWeight}

func roundAtSlot(slot primitives.Slot) uint64 {
	return uint64(slot / params.BeaconConfig().SlotsPerRound)
}

func previousRound(st state.ReadOnlyBeaconState) uint64 {
	r := roundAtSlot(st.Slot())
	if r == 0 {
		return 0
	}
	return r - 1
}

func finalityDelay(st state.ReadOnlyBeaconState) (uint64, error) {
	finalizedSlot, err := st.FinalizedSlot()
	if err != nil {
		return 0, err
	}
	previous, finalized := previousRound(st), roundAtSlot(finalizedSlot)
	if finalized > previous {
		return 0, nil
	}
	return previous - finalized, nil
}

func inInactivityLeak(st state.ReadOnlyBeaconState) (bool, error) {
	delay, err := finalityDelay(st)
	if err != nil {
		return false, err
	}
	return delay > params.BeaconConfig().MinRoundsToInactivityLeak, nil
}

func finalizedEpoch(st state.ReadOnlyBeaconState) (primitives.Epoch, error) {
	finalizedSlot, err := st.FinalizedSlot()
	if err != nil {
		return 0, err
	}
	return slots.ToEpoch(finalizedSlot), nil
}

// helpers.ActiveValidatorIndices errors on an empty active set, which the spec allows.
func activeIndices(st state.ReadOnlyBeaconState, epoch primitives.Epoch) []primitives.ValidatorIndex {
	var indices []primitives.ValidatorIndex
	for idx, val := range st.ValidatorsReadOnlySeq() {
		if helpers.IsActiveValidatorUsingTrie(val, epoch) {
			indices = append(indices, idx)
		}
	}
	return indices
}

func attestationCommittee(active []primitives.ValidatorIndex, epoch primitives.Epoch, index uint64) []primitives.ValidatorIndex {
	n := uint64(len(active))
	perRound := params.BeaconConfig().AttestationCommitteesPerRound
	start, end := n*index/perRound, n*(index+1)/perRound
	committee := make([]primitives.ValidatorIndex, 0, end-start)
	for position := start; position < end; position++ {
		committee = append(committee, active[(position+uint64(epoch))%n])
	}
	return committee
}

func committeeIndices(bits []byte) []uint64 {
	var indices []uint64
	for i := range uint64(len(bits)) * 8 {
		if bitAt(bits, i) {
			indices = append(indices, i)
		}
	}
	return indices
}

func bitAt(bits []byte, i uint64) bool {
	return bits[i/8]&(1<<(i%8)) != 0
}

func slotCommittee(st state.ReadOnlyBeaconState, slot primitives.Slot, domain [4]byte, size uint64) ([]primitives.ValidatorIndex, error) {
	epoch := slots.ToEpoch(slot)
	seed, err := helpers.Seed(st, epoch, domain)
	if err != nil {
		return nil, err
	}
	seed = hash.Hash(append(seed[:], bytesutil.Bytes8(uint64(slot))...))
	// Unshuffling once yields indices[compute_shuffled_index(i)] for every i, far cheaper than one shuffled index per candidate.
	candidates, err := helpers.UnshuffleList(activeIndices(st, epoch), seed)
	if err != nil {
		return nil, err
	}
	total := uint64(len(candidates))
	if total == 0 {
		return nil, errors.New("no active validators to sample a committee from")
	}
	maxBalance := params.BeaconConfig().MaxEffectiveBalanceElectra
	selected := make([]primitives.ValidatorIndex, 0, size)
	var randomBytes [32]byte
	for i := uint64(0); uint64(len(selected)) < size; i++ {
		offset := i % 16 * 2
		if offset == 0 {
			randomBytes = hash.Hash(append(seed[:], bytesutil.Bytes8(i/16)...))
		}
		candidate := candidates[i%total]
		eb, err := st.EffectiveBalanceAtIndex(candidate)
		if err != nil {
			return nil, err
		}
		randomValue := uint64(binary.LittleEndian.Uint16(randomBytes[offset : offset+2]))
		if eb*fieldparams.MaxRandomValueElectra >= maxBalance*randomValue {
			selected = append(selected, candidate)
		}
	}
	return selected, nil
}

func validatorWeight(score uint64) uint64 {
	ejection := params.BeaconConfig().EjectionInactivityScore
	score = min(score, ejection)
	return (ejection - score) * (ejection + score)
}

func hasQuorum(st state.ReadOnlyBeaconState, participation []byte) (bool, error) {
	scores, err := st.InactivityScores()
	if err != nil {
		return false, err
	}
	epoch := time.CurrentEpoch(st)
	var total, participating uint64
	for idx, val := range st.ValidatorsReadOnlySeq() {
		if !helpers.IsActiveValidatorUsingTrie(val, epoch) {
			continue
		}
		if uint64(idx) >= uint64(len(participation)) || uint64(idx) >= uint64(len(scores)) {
			return false, fmt.Errorf("validator %d has no participation or inactivity score", idx)
		}
		weight := validatorWeight(scores[idx])
		total += weight
		if participation[idx] != 0 && !val.Slashed() {
			participating += weight
		}
	}
	return participating > 0 && participating*3 >= total*2, nil
}

func roundBaseReward(effectiveBalance, totalActiveBalance uint64) uint64 {
	increment := params.BeaconConfig().EffectiveBalanceIncrement
	perIncrement := increment * params.BeaconConfig().RoundRewardFactor / math.IntegerSquareRoot(totalActiveBalance)
	return effectiveBalance / increment * perIncrement
}

func verifyAggregate(st state.ReadOnlyBeaconState, indices []uint64, domainType [4]byte, epoch primitives.Epoch, obj ssz.HashRoot, sig []byte) error {
	domain, err := signing.Domain(st.Fork(), epoch, domainType, st.GenesisValidatorsRoot())
	if err != nil {
		return err
	}
	root, err := signing.ComputeSigningRoot(obj, domain)
	if err != nil {
		return err
	}
	pubkeys := make([]bls.PublicKey, len(indices))
	for i, index := range indices {
		if index >= uint64(st.NumValidators()) {
			return fmt.Errorf("validator %d is not in the registry", index)
		}
		key := st.PubkeyAtIndex(primitives.ValidatorIndex(index))
		pubkeys[i], err = bls.PublicKeyFromBytes(key[:])
		if err != nil {
			return errors.Wrapf(err, "validator %d", index)
		}
	}
	signature, err := bls.SignatureFromBytes(sig)
	if err != nil {
		return err
	}
	if !signature.FastAggregateVerify(pubkeys, root) {
		return signing.ErrSigFailedToVerify
	}
	return nil
}
