package eip8453

import (
	"bytes"
	"context"
	"fmt"
	"slices"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/helpers"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/time"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/state"
	"github.com/OffchainLabs/prysm/v7/config/params"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/time/slots"
	"github.com/pkg/errors"
)

// ProcessHeadVote verifies a head vote aggregate and records the seats that voted for the block of their slot.
func ProcessHeadVote(_ context.Context, st state.BeaconState, vote *ethpb.HeadVote) error {
	data := vote.Data
	delay := params.BeaconConfig().MinAttestationInclusionDelay
	if st.Slot() < delay || data.Slot > st.Slot()-delay {
		return fmt.Errorf("head vote slot %d is too recent for state slot %d", data.Slot, st.Slot())
	}
	epoch := slots.ToEpoch(data.Slot)
	current := time.CurrentEpoch(st)
	if epoch != current && epoch != time.PrevEpoch(st) {
		return fmt.Errorf("head vote epoch %d is neither the current nor the previous epoch", epoch)
	}

	committee, err := slotCommittee(st, data.Slot, params.BeaconConfig().DomainHeadVoteCommittee, params.BeaconConfig().HeadVoteCommitteeSize)
	if err != nil {
		return err
	}
	var voters []uint64
	for position, index := range committee {
		if bitAt(vote.AggregationBits, uint64(position)) {
			voters = append(voters, uint64(index))
		}
	}
	if len(voters) == 0 {
		return errors.New("head vote has no voters")
	}
	if err := verifyAggregate(st, voters, params.BeaconConfig().DomainHeadVote, epoch, data, vote.Signature); err != nil {
		return errors.Wrap(err, "invalid head vote signature")
	}

	sameSlot, err := isHeadVoteSameSlot(st, data)
	if err != nil || !sameSlot {
		return err
	}
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	index := uint64(data.Slot % slotsPerEpoch)
	if epoch == current {
		index += uint64(slotsPerEpoch)
	}
	participation, err := st.HeadVoteParticipation()
	if err != nil {
		return err
	}
	participation = slices.Clone(participation)
	seats := slices.Clone(participation[index])
	for position := range params.BeaconConfig().HeadVoteCommitteeSize {
		if bitAt(vote.AggregationBits, position) {
			seats[position/8] |= 1 << (position % 8)
		}
	}
	participation[index] = seats
	return st.SetHeadVoteParticipation(participation)
}

func isHeadVoteSameSlot(st state.ReadOnlyBeaconState, data *ethpb.HeadVoteData) (bool, error) {
	if data.Slot == params.BeaconConfig().GenesisSlot {
		return true, nil
	}
	root, err := helpers.BlockRootAtSlot(st, data.Slot)
	if err != nil {
		return false, err
	}
	if !bytes.Equal(data.BeaconBlockRoot, root) {
		return false, nil
	}
	parentRoot, err := helpers.BlockRootAtSlot(st, data.Slot-1)
	if err != nil {
		return false, err
	}
	return !bytes.Equal(data.BeaconBlockRoot, parentRoot), nil
}
