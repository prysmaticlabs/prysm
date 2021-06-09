package altair

import (
	"bytes"
	"fmt"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// NextSyncCommittee returns the next sync committee for a given state.
//
// Spec code:
// def get_next_sync_committee(state: BeaconState) -> SyncCommittee:
//    """
//    Return the next sync committee, with possible pubkey duplicates.
//    """
//    indices = get_next_sync_committee_indices(state)
//    pubkeys = [state.validators[index].pubkey for index in indices]
//    aggregate_pubkey = bls.AggregatePKs(pubkeys)
//    return SyncCommittee(pubkeys=pubkeys, aggregate_pubkey=aggregate_pubkey)
func NextSyncCommittee(state iface.BeaconStateAltair) (*pb.SyncCommittee, error) {
	indices, err := NextSyncCommitteeIndices(state)
	if err != nil {
		return nil, err
	}
	pubkeys := make([][]byte, params.BeaconConfig().SyncCommitteeSize)
	for i, index := range indices {
		p := state.PubkeyAtIndex(index) // Using ReadOnlyValidators interface. No copy here.
		pubkeys[i] = p[:]
	}
	aggregated, err := bls.AggregatePublicKeys(pubkeys)
	if err != nil {
		return nil, err
	}
	return &pb.SyncCommittee{
		Pubkeys:         pubkeys,
		AggregatePubkey: aggregated.Marshal(),
	}, nil
}

// NextSyncCommitteeIndices returns the next sync committee indices for a given state.
//
// Spec code:
// def get_next_sync_committee_indices(state: BeaconState) -> Sequence[ValidatorIndex]:
//    """
//    Return the sync committee indices, with possible duplicates, for the next sync committee.
//    """
//    epoch = Epoch(get_current_epoch(state) + 1)
//
//    MAX_RANDOM_BYTE = 2**8 - 1
//    active_validator_indices = get_active_validator_indices(state, epoch)
//    active_validator_count = uint64(len(active_validator_indices))
//    seed = get_seed(state, epoch, DOMAIN_SYNC_COMMITTEE)
//    i = 0
//    sync_committee_indices: List[ValidatorIndex] = []
//    while len(sync_committee_indices) < SYNC_COMMITTEE_SIZE:
//        shuffled_index = compute_shuffled_index(uint64(i % active_validator_count), active_validator_count, seed)
//        candidate_index = active_validator_indices[shuffled_index]
//        random_byte = hash(seed + uint_to_bytes(uint64(i // 32)))[i % 32]
//        effective_balance = state.validators[candidate_index].effective_balance
//        if effective_balance * MAX_RANDOM_BYTE >= MAX_EFFECTIVE_BALANCE * random_byte:
//            sync_committee_indices.append(candidate_index)
//        i += 1
//    return sync_committee_indices
func NextSyncCommitteeIndices(state iface.BeaconStateAltair) ([]types.ValidatorIndex, error) {
	epoch := helpers.NextEpoch(state)
	indices, err := helpers.ActiveValidatorIndices(state, epoch)
	if err != nil {
		return nil, err
	}
	count := uint64(len(indices))
	seed, err := helpers.Seed(state, epoch, params.BeaconConfig().DomainSyncCommittee)
	if err != nil {
		return nil, err
	}
	i := types.ValidatorIndex(0)
	cIndices := make([]types.ValidatorIndex, 0, params.BeaconConfig().SyncCommitteeSize)
	hashFunc := hashutil.CustomSHA256Hasher()
	maxRandomByte := uint64(1<<8 - 1)
	for uint64(len(cIndices)) < params.BeaconConfig().SyncCommitteeSize {
		sIndex, err := helpers.ComputeShuffledIndex(i.Mod(count), count, seed, true)
		if err != nil {
			return nil, err
		}

		b := append(seed[:], bytesutil.Bytes8(uint64(i.Div(32)))...)
		randomByte := hashFunc(b)[i%32]
		cIndex := indices[sIndex]
		v, err := state.ValidatorAtIndexReadOnly(cIndex)
		if err != nil {
			return nil, err
		}

		effectiveBal := v.EffectiveBalance()
		if effectiveBal*maxRandomByte >= params.BeaconConfig().MaxEffectiveBalance*uint64(randomByte) {
			cIndices = append(cIndices, cIndex)
		}
		i++
	}
	return cIndices, nil
}

// AssignedToSyncCommittee returns true if input validator `i` is assigned to a sync committee at epoch `e`.
//
// Spec code:
// def is_assigned_to_sync_committee(state: BeaconState,
//                                  epoch: Epoch,
//                                  validator_index: ValidatorIndex) -> bool:
//    sync_committee_period = compute_sync_committee_period(epoch)
//    current_epoch = get_current_epoch(state)
//    current_sync_committee_period = compute_sync_committee_period(current_epoch)
//    next_sync_committee_period = current_sync_committee_period + 1
//    assert sync_committee_period in (current_sync_committee_period, next_sync_committee_period)
//
//    pubkey = state.validators[validator_index].pubkey
//    if sync_committee_period == current_sync_committee_period:
//        return pubkey in state.current_sync_committee.pubkeys
//    else:  # sync_committee_period == next_sync_committee_period
//        return pubkey in state.next_sync_committee.pubkeys
func AssignedToSyncCommittee(
	state iface.BeaconStateAltair,
	epoch types.Epoch,
	i types.ValidatorIndex,
) (bool, error) {
	p := SyncCommitteePeriod(epoch)
	currentEpoch := helpers.CurrentEpoch(state)
	currentPeriod := SyncCommitteePeriod(currentEpoch)
	nextEpoch := currentPeriod + 1

	if p != currentPeriod && p != nextEpoch {
		return false, fmt.Errorf("epoch period %d is not current period %d or next period %d in state", p, currentEpoch, nextEpoch)
	}

	v, err := state.ValidatorAtIndexReadOnly(i)
	if err != nil {
		return false, err
	}
	vPubKey := v.PublicKey()

	hasKey := func(c *pb.SyncCommittee, k [48]byte) (bool, error) {
		for _, p := range c.Pubkeys {
			if bytes.Equal(vPubKey[:], p) {
				return true, nil
			}
		}
		return false, nil
	}

	if p == currentPeriod {
		c, err := state.CurrentSyncCommittee()
		if err != nil {
			return false, err
		}
		return hasKey(c, vPubKey)
	}

	c, err := state.NextSyncCommittee()
	if err != nil {
		return false, err
	}
	return hasKey(c, vPubKey)
}

// SubnetsForSyncCommittee returns subnet number of what validator `i` belongs to.
//
// Spec code:
// def compute_subnets_for_sync_committee(state: BeaconState, validator_index: ValidatorIndex) -> Sequence[uint64]:
//    next_slot_epoch = compute_epoch_at_slot(Slot(state.slot + 1))
//    if compute_sync_committee_period(get_current_epoch(state)) == compute_sync_committee_period(next_slot_epoch):
//        sync_committee = state.current_sync_committee
//    else:
//        sync_committee = state.next_sync_committee
//
//    target_pubkey = state.validators[validator_index].pubkey
//    sync_committee_indices = [index for index, pubkey in enumerate(sync_committee.pubkeys) if pubkey == target_pubkey]
//    return [
//        uint64(index // (SYNC_COMMITTEE_SIZE // SYNC_COMMITTEE_SUBNET_COUNT))
//        for index in sync_committee_indices
//    ]
func SubnetsForSyncCommittee(state iface.BeaconStateAltair, i types.ValidatorIndex) ([]uint64, error) {
	committee, err := state.NextSyncCommittee()
	if err != nil {
		return nil, err
	}

	nextSlotEpoch := helpers.SlotToEpoch(state.Slot() + 1)
	if SyncCommitteePeriod(nextSlotEpoch) == SyncCommitteePeriod(helpers.CurrentEpoch(state)) {
		committee, err = state.CurrentSyncCommittee()
		if err != nil {
			return nil, err
		}
	}

	v, err := state.ValidatorAtIndexReadOnly(i)
	if err != nil {
		return nil, err
	}
	vPubKey := v.PublicKey()

	positions := make([]uint64, 0)
	for i, pkey := range committee.Pubkeys {
		if bytes.Equal(vPubKey[:], pkey) {
			positions = append(positions, uint64(i)/(params.BeaconConfig().SyncCommitteeSize/params.BeaconConfig().SyncCommitteeSubnetCount))
		}
	}
	return positions, nil
}

// SyncCommitteePeriod returns the sync committee period of input epoch `e`.
//
// Spec code:
// def compute_sync_committee_period(epoch: Epoch) -> uint64:
//    return epoch // EPOCHS_PER_SYNC_COMMITTEE_PERIOD
func SyncCommitteePeriod(epoch types.Epoch) uint64 {
	return uint64(epoch / params.BeaconConfig().EpochsPerSyncCommitteePeriod)
}
