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

// NextSyncCommittee returns the next sync committee for a given state.
//
// Spec code:
// def get_next_sync_committee(state: BeaconState) -> SyncCommittee:
//    """
//    Return the *next* sync committee for a given ``state``.
//
//    ``SyncCommittee`` contains an aggregate pubkey that enables
//    resource-constrained clients to save some computation when verifying
//    the sync committee's signature.
//
//    ``SyncCommittee`` can also contain duplicate pubkeys, when ``get_next_sync_committee_indices``
//    returns duplicate indices. Implementations must take care when handling
//    optimizations relating to aggregation and verification in the presence of duplicates.
//
//    Note: This function should only be called at sync committee period boundaries by ``process_sync_committee_updates``
//    as ``get_next_sync_committee_indices`` is not stable within a given period.
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
//    Return the sequence of sync committee indices (which may include duplicate indices)
//    for the next sync committee, given a ``state`` at a sync committee period boundary.
//
//    Note: Committee can contain duplicate indices for small validator sets (< SYNC_COMMITTEE_SIZE + 128)
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
