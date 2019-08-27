package core

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/light/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// UnpackCompactValidator unpacks the compact validator object into index, slashed and balance forms.
//
// Spec pseudocode definition:
//   def unpack_compact_validator(compact_validator: CompactValidator) -> Tuple[ValidatorIndex, bool, uint64]:
//    """
//    Return the index, slashed, effective_balance // EFFECTIVE_BALANCE_INCREMENT of ``compact_validator``.
//    """
//    return (
//        ValidatorIndex(compact_validator >> 16),
//        (compact_validator >> 15) % 2,
//        uint64(compact_validator & (2**15 - 1)),
//    )
func UnpackCompactValidator(compactValidator uint64) (uint64, bool, uint64) {
	index := compactValidator >> 16
	slashed := (compactValidator >> 15) % 2 == 0
	balance := uint64(compactValidator & (1 << 15 -1))

	return index, slashed, balance
}

// PersistentCommitteePubkeysBalances returns the public keys and the balance of the
// persistent committee at input epoch.
//
// Spec pseudocode definition:
//   def get_persistent_committee_pubkeys_and_balances(memory: LightClientMemory,
//                                                  epoch: Epoch) -> Tuple[Sequence[BLSPubkey], Sequence[uint64]]:
//    """
//    Return pubkeys and balances for the persistent committee at ``epoch``.
//    """
//    current_period = compute_epoch_of_slot(memory.header.slot) // EPOCHS_PER_SHARD_PERIOD
//    next_period = epoch // EPOCHS_PER_SHARD_PERIOD
//    assert next_period in (current_period, current_period + 1)
//    if next_period == current_period:
//        earlier_committee, later_committee = memory.previous_committee, memory.current_committee
//    else:
//        earlier_committee, later_committee = memory.current_committee, memory.next_committee
//
//    pubkeys = []
//    balances = []
//    for pubkey, compact_validator in zip(earlier_committee.pubkeys, earlier_committee.compact_validators):
//        index, slashed, balance = unpack_compact_validator(compact_validator)
//        if epoch % EPOCHS_PER_SHARD_PERIOD < index % EPOCHS_PER_SHARD_PERIOD:
//            pubkeys.append(pubkey)
//            balances.append(balance)
//    for pubkey, compact_validator in zip(later_committee.pubkeys, later_committee.compact_validators):
//        index, slashed, balance = unpack_compact_validator(compact_validator)
//        if epoch % EPOCHS_PER_SHARD_PERIOD >= index % EPOCHS_PER_SHARD_PERIOD:
//            pubkeys.append(pubkey)
//            balances.append(balance)
//    return pubkeys, balances
func PersistentCommitteePubkeysBalances(store *pb.LightClientStore, epoch uint64) ([][]byte, []uint64){
	currentEpoch := helpers.SlotToEpoch(store.Header.Slot) / params.BeaconConfig().EpochsPerShardPeriod
	nextEpoch := epoch / params.BeaconConfig().EpochsPerShardPeriod
	return nil, nil
}