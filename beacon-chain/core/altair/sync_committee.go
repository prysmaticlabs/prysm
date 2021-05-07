package altair

import (
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// SyncCommittee returns the sync committee for a given state and epoch.
//
// Spec code:
// def get_sync_committee(state: BeaconState, epoch: Epoch) -> SyncCommittee:
//    """
//    Return the sync committee for a given state and epoch.
//    """
//    indices = get_sync_committee_indices(state, epoch)
//    pubkeys = [state.validators[index].pubkey for index in indices]
//    partition = [pubkeys[i:i + SYNC_PUBKEYS_PER_AGGREGATE] for i in range(0, len(pubkeys), SYNC_PUBKEYS_PER_AGGREGATE)]
//    pubkey_aggregates = [bls.AggregatePKs(preaggregate) for preaggregate in partition]
//    return SyncCommittee(pubkeys=pubkeys, pubkey_aggregates=pubkey_aggregates)
func SyncCommittee(state iface.BeaconStateAltair, epoch types.Epoch) (*pb.SyncCommittee, error) {
	indices, err := SyncCommitteeIndices(state, epoch)
	if err != nil {
		return nil, err
	}
	pubkeys := make([][]byte, params.BeaconConfig().SyncCommitteeSize)
	for i, index := range indices {
		p := state.PubkeyAtIndex(index) // Using ReadOnlyValidators interface. No copy here.
		pubkeys[i] = p[:]
	}
	aggregates := make([][]byte, 0, params.BeaconConfig().SyncPubkeysPerAggregate)
	for i := uint64(0); i < uint64(len(pubkeys)); i += params.BeaconConfig().SyncPubkeysPerAggregate {
		a, err := bls.AggregatePublicKeys(pubkeys[i : i+params.BeaconConfig().SyncPubkeysPerAggregate])
		if err != nil {
			return nil, err
		}
		aggregates = append(aggregates, a.Marshal())
	}
	return &pb.SyncCommittee{
		Pubkeys:          pubkeys,
		PubkeyAggregates: aggregates,
	}, nil
}

// SyncCommitteeIndices returns the sync committee indices for a given state and epoch.
//
// Spec code:
// def get_sync_committee_indices(state: BeaconState, epoch: Epoch) -> Sequence[ValidatorIndex]:
//    """
//    Return the sequence of sync committee indices (which may include duplicate indices) for a given state and epoch.
//    """
//    MAX_RANDOM_BYTE = 2**8 - 1
//    base_epoch = Epoch((max(epoch // EPOCHS_PER_SYNC_COMMITTEE_PERIOD, 1) - 1) * EPOCHS_PER_SYNC_COMMITTEE_PERIOD)
//    active_validator_indices = get_active_validator_indices(state, base_epoch)
//    active_validator_count = uint64(len(active_validator_indices))
//    seed = get_seed(state, base_epoch, DOMAIN_SYNC_COMMITTEE)
//    i = 0
//    sync_committee_indices: List[ValidatorIndex] = []
//    while len(sync_committee_indices) < SYNC_COMMITTEE_SIZE:
//        shuffled_index = compute_shuffled_index(uint64(i % active_validator_count), active_validator_count, seed)
//        candidate_index = active_validator_indices[shuffled_index]
//        random_byte = hash(seed + uint_to_bytes(uint64(i // 32)))[i % 32]
//        effective_balance = state.validators[candidate_index].effective_balance
//        if effective_balance * MAX_RANDOM_BYTE >= MAX_EFFECTIVE_BALANCE * random_byte:  # Sample with replacement
//            sync_committee_indices.append(candidate_index)
//        i += 1
//    return sync_committee_indices
func SyncCommitteeIndices(state iface.BeaconStateAltair, epoch types.Epoch) ([]types.ValidatorIndex, error) {
	e := types.Epoch(1)
	p := params.BeaconConfig().EpochsPerSyncCommitteePeriod
	syncPeriod := epoch / p
	if syncPeriod > 1 {
		e = syncPeriod
	}

	baseEpoch := e.Sub(1) * p

	indices, err := helpers.ActiveValidatorIndices(state, baseEpoch)
	if err != nil {
		return nil, err
	}
	count := uint64(len(indices))
	seed, err := helpers.Seed(state, baseEpoch, params.BeaconConfig().DomainSyncCommittee)
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
