package transition

import (
	"bytes"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	beaconHelper "github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	shardHelper "github.com/prysmaticlabs/prysm/shard-chains/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// ProcessShardBlock processes block of a shard.
//
// Spec pseudocode definition:
//  def process_shard_block(state: BeaconState, shard_state: ShardState, block: ShardBlock) -> None:
//    process_shard_block_header(state, shard_state, block)
//    process_shard_attestations(state, shard_state, block)
//    process_shard_block_size_fee(state, shard_state, block)
func ProcessShardBlock(beaconState *pb.BeaconState, shardState *ethpb.ShardState, shardBlock *ethpb.ShardBlock) (*ethpb.ShardState, error) {
	var err error
	shardState, err = ProcessShardBlockHeader(beaconState, shardState, shardBlock)
	if err != nil {
		return nil, errors.Wrap(err, "could not process shard block header")
	}
	shardState, err = ProcessShardAttestations(beaconState, shardState, shardBlock)
	if err != nil {
		return nil, errors.Wrap(err, "could not process shard attestations")
	}

	shardState, err = ProcessShardBlockSizeFee(beaconState, shardState, shardBlock)
	if err != nil {
		return nil, errors.Wrap(err, "could not process shard block size fee")
	}

	return shardState, err
}

// ProcessShardBlockHeader processes block header of a shard block.
//
// Spec pseudocode definition:
//  def process_shard_block_header(state: BeaconState, shard_state: ShardState, block: ShardBlock) -> None:
//    # Verify that the slots match
//    data = block.data
//    assert data.slot == state.slot
//    # Verify that the beacon chain root matches
//    parent_epoch = compute_epoch_of_shard_slot(state.latest_block_header_data.slot)
//    assert data.beacon_block_root == get_block_root(state, parent_epoch)
//    # Verify that the parent matches
//    assert data.parent_root == hash_tree_root(state.latest_block_header_data)
//    # Save current block as the new latest block
//    state.latest_block_header_data = ShardBlockHeaderData(
//        slot=data.slot,
//        beacon_block_root=data.beacon_block_root,
//        parent_root=data.parent_root,
//        # `state_root` is zeroed and overwritten in the next `process_shard_slot` call
//        aggregation_bits=data.aggregation_bits,
//        block_size_sum=data.block_size_sum,
//        body_root=hash_tree_root(data.body),
//    )
//    # Verify proposer signature
//    proposer_index = get_shard_proposer_index(state, state.shard, data.slot)
//    pubkey = state.validators[proposer_index].pubkey
//    domain = get_domain(state, DOMAIN_SHARD_PROPOSER, compute_epoch_of_shard_slot(data.slot))
//    assert bls_verify(pubkey, hash_tree_root(block.data), block.signatures.proposer, domain)
//    # Verify body size is a multiple of the header size
//    assert len(data.body) % SHARD_HEADER_SIZE == 0
//    # Verify the sum of the block sizes since genesis
//    state.block_size_sum += SHARD_HEADER_SIZE + len(data.body)
//    assert data.block_size_sum == state.block_size_sum
func ProcessShardBlockHeader(beaconState *pb.BeaconState, shardState *ethpb.ShardState, shardBlock *ethpb.ShardBlock) (*ethpb.ShardState, error) {
	// Verify shard slots match
	data := shardBlock.Data
	if shardState.Slot != data.Slot {
		return nil, fmt.Errorf("shard state slot: %d is different then shard block slot: %d", shardState.Slot, data.Slot)
	}

	// Verify beacon chain root matches
	parentEpoch := shardHelper.ShardSlotToEpoch(shardState.LatestBlockHeaderData.Slot)
	beaconBlockRoot, err := beaconHelper.BlockRoot(beaconState, parentEpoch)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(data.BeaconBlockRoot, beaconBlockRoot[:]) {
		return nil, fmt.Errorf(
			"beacon block root %#x does not match the beacon block root in state %#x",
			data.BeaconBlockRoot, beaconBlockRoot)
	}

	// Verify shard block parent root matches
	parentRoot, err := ssz.HashTreeRoot(shardState.LatestBlockHeaderData)
	if err != nil {
		return nil, errors.Wrap(err, "could hash tree root shard block header")
	}
	if !bytes.Equal(data.ParentRoot, parentRoot[:]) {
		return nil, fmt.Errorf(
			"shard parent root %#x does not match the latest block header hash tree root in state %#x",
			data.ParentRoot, parentRoot)
	}

	// Save current shard block as latest block in state
	bodyRoot, err := ssz.HashTreeRoot(data.Body)
	if err != nil {
		return nil, errors.Wrap(err, "could not tree hash shard block body")
	}
	shardState.LatestBlockHeaderData = &ethpb.ShardBlockHeaderData{
		Slot:            data.Slot,
		BeaconBlockRoot: data.BeaconBlockRoot,
		ParentRoot:      data.ParentRoot,
		AggregationBits: data.AggregationBits,
		BlockSizeSum:    data.BlockSizeSum,
		BodyRoot:        bodyRoot[:],
	}

	// Verify proposer index

	// Verify body size is a multiple of header size
	if uint64(len(data.Body))%params.BeaconConfig().ShardHeaderSize != 0 {
		return nil, fmt.Errorf("body size %d is not a multiple of header size %d",
			len(data.Body), params.BeaconConfig().ShardHeaderSize)
	}

	// Verify sum of block sizes since genesis
	sum := shardState.BlockSizeSum + uint64(len(data.Body)) + params.BeaconConfig().ShardHeaderSize
	if data.BlockSizeSum != sum {
		return nil, fmt.Errorf("body size %d is not equal to block size in state %d",
			data.BlockSizeSum, sum)
	}

	return shardState, nil
}

// ProcessShardAttestations processes attestations for a shard block
//
// Spec pseudocode definition:
//  def process_shard_attestations(state: BeaconState, shard_state: ShardState, block: ShardBlock) -> None:
//    data = block.data
//    pubkeys = []
//    attestation_count = 0
//    shard_committee = get_shard_committee(state, state.shard, data.slot)
//    for i, validator_index in enumerate(shard_committee):
//        if data.aggregation_bits[i]:
//            pubkeys.append(state.validators[validator_index].pubkey)
//            add_reward(state, shard_state, validator_index, get_base_reward(state, validator_index))
//            attestation_count += 1
//    # Verify there are no extraneous bits set beyond the shard committee
//    for i in range(len(shard_committee), 2 * MAX_PERIOD_COMMITTEE_SIZE):
//        assert data.aggregation_bits[i] == 0b0
//    # Verify attester aggregate signature
//    domain = get_domain(state, DOMAIN_SHARD_ATTESTER, compute_epoch_of_shard_slot(data.slot))
//    message = hash_tree_root(ShardCheckpoint(shard_state.slot, data.parent_root))
//    assert bls_verify(bls_aggregate_pubkeys(pubkeys), message, block.signatures.attesters, domain)
//    # Proposer micro-reward
//    proposer_index = get_shard_proposer_index(state, state.shard, data.slot)
//    reward = attestation_count * get_base_reward(state, proposer_index) // PROPOSER_REWARD_QUOTIENT
//    add_reward(state, shard_state, proposer_index, reward)
func ProcessShardAttestations(beaconState *pb.BeaconState, shardState *ethpb.ShardState, shardBlock *ethpb.ShardBlock) (*ethpb.ShardState, error) {
	data := shardBlock.Data
	shardCommittee, err := shardHelper.ShardCommittee(beaconState, shardState.Shard, data.Slot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get shard commitee")
	}
	pubKeys := make([][]byte, 0, len(shardCommittee))
	attCount := uint64(0)
	for i, validatorIdx := range shardCommittee {
		if data.AggregationBits[i] == 1 {
			pubKeys = append(pubKeys, beaconState.Validators[validatorIdx].PublicKey)
			baseReward, err := beaconHelper.BaseReward(beaconState, validatorIdx)
			if err != nil {
				return nil, errors.Wrapf(err, "could not get base reward for validator index %d", validatorIdx)
			}
			shardState, err = shardHelper.AddReward(beaconState, shardState, validatorIdx, baseReward)
			if err != nil {
				return nil, errors.Wrapf(err, "could not add reward for validator index %d", validatorIdx)
			}
			attCount++
		}
	}
	// Verify there are no extraneous bits set beyond the shard committee
	start := uint64(len(shardCommittee))
	end := 2 * params.BeaconConfig().MaxPeriodCommitteeSize
	for i := start; i < end; i++ {
		if data.AggregationBits.BitAt(i) {
			return nil, fmt.Errorf("aggregation bit at index %d should not have been set", i)
		}
	}

	// Verify attester's aggregate signature

	// Handle proposer micro reward
	proposerIdx, err := shardHelper.ShardProposerIndex(beaconState, shardState.Shard, data.Slot)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get shard proposer index")
	}
	baseReward, err := beaconHelper.BaseReward(beaconState, proposerIdx)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get base reward for proposer index %d", proposerIdx)
	}
	reward := attCount * baseReward / params.BeaconConfig().ProposerRewardQuotient
	shardState, err = shardHelper.AddReward(beaconState, shardState, proposerIdx, reward)
	if err != nil {
		return nil, errors.Wrapf(err, "could not add reward for proposer index %d", proposerIdx)
	}

	return shardState, nil
}

// ProcessShardBlockSizeFee processes the block fee based on the size of the block.
//
// Spec pseudocode definition:
//  def process_shard_block_size_fee(state: BeaconState, shard_state: ShardState, block: ShardBlock) -> None:
//    # Charge proposer block size fee
//    proposer_index = get_shard_proposer_index(state, state.shard, block.data.slot)
//    block_size = SHARD_HEADER_SIZE + len(block.data.body)
//    add_fee(state, shard_state, proposer_index, state.block_size_price * block_size // SHARD_BLOCK_SIZE_LIMIT)
//    # Calculate new block size price
//    if block_size > SHARD_BLOCK_SIZE_TARGET:
//        size_delta = block_size - SHARD_BLOCK_SIZE_TARGET
//        price_delta = Gwei(state.block_size_price * size_delta // SHARD_BLOCK_SIZE_LIMIT // BLOCK_SIZE_PRICE_QUOTIENT)
//        # The maximum gas price caps the amount burnt on gas fees within a period to 32 ETH
//        MAX_BLOCK_SIZE_PRICE = MAX_EFFECTIVE_BALANCE // EPOCHS_PER_SHARD_PERIOD // SHARD_SLOTS_PER_EPOCH
//        state.block_size_price = min(MAX_BLOCK_SIZE_PRICE, state.block_size_price + price_delta)
//    else:
//        size_delta = SHARD_BLOCK_SIZE_TARGET - block_size
//        price_delta = Gwei(state.block_size_price * size_delta // SHARD_BLOCK_SIZE_LIMIT // BLOCK_SIZE_PRICE_QUOTIENT)
//        state.block_size_price = max(MIN_BLOCK_SIZE_PRICE, state.block_size_price - price_delta)
func ProcessShardBlockSizeFee(beaconState *pb.BeaconState, shardState *ethpb.ShardState, shardBlock *ethpb.ShardBlock) (*ethpb.ShardState, error) {
	// Charge proposer block size fee
	proposerIdx, err := shardHelper.ShardProposerIndex(beaconState, shardState.Shard, shardBlock.Data.Slot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get proposer index")
	}
	blockSize := uint64(len(shardBlock.Data.Body)) + params.BeaconConfig().ShardHeaderSize
	fee := shardState.BlockSizePrice * blockSize / params.BeaconConfig().ShardBlockSizeLimit
	shardState, err = shardHelper.AddFee(beaconState, shardState, proposerIdx, fee)
	if err != nil {
		return nil, errors.Wrap(err, "could not add fee to proposer")
	}

	// Calculate and change new block size pricing
	if blockSize > params.BeaconConfig().ShardBlockSizeTarget {
		sizeDelta := blockSize - params.BeaconConfig().ShardBlockSizeTarget
		priceDelta := shardState.BlockSizePrice * sizeDelta / params.BeaconConfig().ShardBlockSizeLimit / params.BeaconConfig().BlockSizeQuotient
		// Max gas price caps the amount burnt on gas fees within a period to 32ETH
		maxBlockSizePrice := params.BeaconConfig().MaxEffectiveBalance / params.BeaconConfig().EpochsPerShardPeriod / params.BeaconConfig().ShardSlotsPerEpoch
		if shardState.BlockSizePrice+priceDelta < maxBlockSizePrice {
			shardState.BlockSizePrice = shardState.BlockSizePrice + priceDelta
		} else {
			shardState.BlockSizePrice = maxBlockSizePrice
		}
		return shardState, nil
	}

	sizeDelta := params.BeaconConfig().ShardBlockSizeTarget - blockSize
	priceDelta := shardState.BlockSizePrice * sizeDelta / params.BeaconConfig().ShardBlockSizeLimit / params.BeaconConfig().BlockSizeQuotient
	if shardState.BlockSizePrice-priceDelta > params.BeaconConfig().MinBlockSizePrice {
		shardState.BlockSizePrice = shardState.BlockSizePrice - priceDelta
	} else {
		shardState.BlockSizePrice = params.BeaconConfig().MinBlockSizePrice
	}

	return shardState, nil
}
