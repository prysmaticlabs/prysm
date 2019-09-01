package light

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/light/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// Client represents a service struct that handles the light client
// logic in eth2.0.
type Client struct {
	ctx              context.Context
	cancel           context.CancelFunc
	db               db.Database
	store            *pb.LightClientStore
}

// NewClient instantiates a new light client service instance.
func NewClient(ctx context.Context, db db.Database) *Client {
	ctx, cancel := context.WithCancel(ctx)
	return &Client{
		ctx:             ctx,
		cancel:          cancel,
		db:              db,
		store:           &pb.LightClientStore{},
	}
}


// UpdateStore updates light client's state in memory.
//
// Spec pseudocode definition:
//   def update_memory(memory: LightClientMemory, update: LightClientUpdate) -> None:
//    # Verify the update does not skip a period
//    current_period = compute_epoch_of_slot(memory.header.slot) // EPOCHS_PER_SHARD_PERIOD
//    next_epoch = compute_epoch_of_shard_slot(update.header.slot)
//    next_period = next_epoch // EPOCHS_PER_SHARD_PERIOD
//    assert next_period in (current_period, current_period + 1)
//
//    # Verify update header against shard block root and header branch
//    assert is_valid_merkle_branch(
//        leaf=hash_tree_root(update.header),
//        branch=update.header_branch,
//        depth=BEACON_CHAIN_ROOT_IN_SHARD_BLOCK_HEADER_DEPTH,
//        index=BEACON_CHAIN_ROOT_IN_SHARD_BLOCK_HEADER_INDEX,
//        root=update.shard_block_root,
//    )
//
//    # Verify persistent committee votes pass 2/3 threshold
//    pubkeys, balances = get_persistent_committee_pubkeys_and_balances(memory, next_epoch)
//    assert 3 * sum(filter(lambda i: update.aggregation_bits[i], balances)) > 2 * sum(balances)
//
//    # Verify shard attestations
//    pubkey = bls_aggregate_pubkeys(filter(lambda i: update.aggregation_bits[i], pubkeys))
//    domain = compute_domain(DOMAIN_SHARD_ATTESTER, update.fork_version)
//    assert bls_verify(pubkey, update.shard_block_root, update.signature, domain)
//
//    # Update persistent committees if entering a new period
//    if next_period == current_period + 1:
//        assert is_valid_merkle_branch(
//            leaf=hash_tree_root(update.committee),
//            branch=update.committee_branch,
//            depth=PERSISTENT_COMMITTEE_ROOT_IN_BEACON_STATE_DEPTH + log_2(SHARD_COUNT),
//            index=PERSISTENT_COMMITTEE_ROOT_IN_BEACON_STATE_INDEX << log_2(SHARD_COUNT) + memory.shard,
//            root=hash_tree_root(update.header),
//        )
//        memory.previous_committee = memory.current_committee
//        memory.current_committee = memory.next_committee
//        memory.next_committee = update.committee
//
//    # Update header
//    memory.header = update.header
func (c *Client) UpdateStore(update *pb.LightClientUpdate) error {
	// Verify update does not skip a period
	epochsPerShardPeriod := params.BeaconConfig().EpochsPerShardPeriod
	currentPeriod := helpers.SlotToEpoch(c.store.Header.Slot) / epochsPerShardPeriod
	nextEpoch := helpers.SlotToEpoch(update.Header.Slot)
	nextPeriod := nextEpoch / epochsPerShardPeriod

	if nextPeriod != currentPeriod || nextPeriod != currentPeriod + 1 {
		return fmt.Errorf("next epoch not equal to current epoch or plus one, %d != (%d, %d)",
			nextPeriod, currentPeriod, currentPeriod + 1)
	}

	// TODO: Verify updated block header against shard block root and block header branch

	// Verify persistent committee votes pass 2/3 threshold
	pubkeys, balances, err := PersistentCommitteePubkeysBalances(c.store, nextEpoch)
	if err != nil {
		return errors.Wrap(err, "could not get persistent committee votes")
	}
	votedBalances := uint64(0)
	validatorIndex := 0
	for _, b := range update.AggregationBits {
		for i := uint64(0); i < 7 ; i++ {
			if bytesutil.IsBitSet(b, i) {
				votedBalances += balances[validatorIndex]
			}
			validatorIndex++
			if validatorIndex == len(pubkeys) {
				break
			}
		}
	}

	totalBalance := uint64(0)
	for _, b := range balances {
		totalBalance += b
	}

	if 3 * votedBalances <= 2 * totalBalance {
		return errors.Wrapf(err, "committee votes did not pass 2/3 threshold, %d <= %d",
			3 * votedBalances, 2 * totalBalance)
	}

	// TODO: Verify shard attestation signature

	// Update persistent committee in store when entering a new period
	if nextPeriod == currentPeriod + 1 {
		// TODO: Verify updated committee against block header and committee branch

		c.store.PreviousCommittee = c.store.CurrentCommittee
		c.store.CurrentCommittee = c.store.NextCommittee
		c.store.NextCommittee = update.Committee
	}

	c.store.Header = update.Header

	return nil
}
