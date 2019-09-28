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
//    # Verify the shard number
//    assert block.shard == state.shard
//    # Verify the slot number
//    assert block.slot == state.slot
//    # Verify the beacon chain root
//    parent_epoch = compute_epoch_of_shard_slot(state.latest_block_header.slot)
//    assert block.beacon_block_root == get_block_root(state, parent_epoch)
//    # Verify the parent root
//    assert block.parent_root == hash_tree_root(state.latest_block_header)
//    # Save current block as the new latest block
//    state.latest_block_header = ShardBlockHeader(
//        shard=block.shard,
//        slot=block.slot,
//        beacon_block_root=block.beacon_block_root,
//        parent_root=block.parent_root,
//        # `state_root` is zeroed and overwritten in the next `process_shard_slot` call
//        block_size_sum=block.block_size_sum,
//        body_root=hash_tree_root(block.body),
//        aggregation_bits=block.aggregation_bits,
//        attestations=block.attestations,
//        # `signature` is zeroed
//    )
//    # Verify the sum of the block sizes since genesis
//    state.block_size_sum += SHARD_HEADER_SIZE + len(block.body)
//    assert block.block_size_sum == state.block_size_sum
//    # Verify proposer signature
//    proposer_index = get_shard_proposer_index(state, state.shard, block.slot)
//    pubkey = state.validators[proposer_index].pubkey
//    domain = get_domain(state, DOMAIN_SHARD_PROPOSER, compute_epoch_of_shard_slot(block.slot))
//    assert bls_verify(pubkey, hash_tree_root(block.block), block.proposer, domain)
func ProcessShardBlockHeader(beaconState *pb.BeaconState, shardState *ethpb.ShardState, shardBlock *ethpb.ShardBlock) (*ethpb.ShardState, error) {
	// Verify shards match
	if shardState.Shard != shardBlock.Shard {
		return nil, fmt.Errorf("shard in state: %d is different then shard in block: %d", shardState.Shard, shardBlock.Shard)
	}

	// Verify slots match
	if shardState.Slot != shardBlock.Slot {
		return nil, fmt.Errorf("shard state slot: %d is different then shard block slot: %d", shardState.Slot, shardBlock.Slot)
	}

	// Verify beacon chain root matches
	parentEpoch := shardHelper.ShardSlotToEpoch(shardState.LatestBlockHeader.Slot)
	beaconBlockRoot, err := beaconHelper.BlockRoot(beaconState, parentEpoch)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(shardBlock.BeaconBlockRoot, beaconBlockRoot[:]) {
		return nil, fmt.Errorf(
			"beacon block root %#x does not match the beacon block root in state %#x",
			shardBlock.BeaconBlockRoot, beaconBlockRoot)
	}

	// Verify shard block parent root matches
	parentRoot, err := ssz.HashTreeRoot(shardState.LatestBlockHeader)
	if err != nil {
		return nil, errors.Wrap(err, "could hash tree root shard block header")
	}
	if !bytes.Equal(shardBlock.ParentRoot, parentRoot[:]) {
		return nil, fmt.Errorf(
			"shard parent root %#x does not match the latest block header hash tree root in state %#x",
			shardBlock.ParentRoot, parentRoot)
	}

	// Save current shard block as latest block in state
	bodyRoot, err := ssz.HashTreeRoot(shardBlock.Body)
	if err != nil {
		return nil, errors.Wrap(err, "could not tree hash shard block body")
	}
	shardState.LatestBlockHeader = &ethpb.ShardBlockHeader{
		Shard:                 shardBlock.Shard,
		Slot:                  shardBlock.Slot,
		BeaconBlockRoot:       shardBlock.BeaconBlockRoot,
		ParentRoot:            shardBlock.ParentRoot,
		AggregationBits:       shardBlock.AggregationBits,
		BlockSizeSum:          shardBlock.BlockSizeSum,
		BodyRoot:              bodyRoot[:],
		AttestationsSignature: shardBlock.AttestationsSignature,
	}

	// Verify sum of block sizes since genesis
	sum := shardState.BlockSizeSum + uint64(len(shardBlock.Body)) + params.ShardConfig().ShardHeaderSize
	if shardBlock.BlockSizeSum != sum {
		return nil, fmt.Errorf("body size %d is not equal to block size in state %d",
			shardBlock.BlockSizeSum, sum)
	}

	// Verify proposer signature

	return shardState, nil
}

// ProcessShardAttestations processes attestations for a shard block
//
// Spec pseudocode definition:
//  def process_shard_attestations(state: BeaconState, shard_state: ShardState, block: ShardBlock) -> None:
//	    pubkeys = []
//	    attestation_count = 0
//	    shard_committee = get_shard_committee(state, state.shard, block.slot)
//	    for i, validator_index in enumerate(shard_committee):
//	        if block.aggregation_bits[i]:
//	            pubkeys.append(state.validators[validator_index].pubkey)
//	            process_delta(state, shard_state, validator_index, get_base_reward(state, validator_index))
//	            attestation_count += 1
//	    # Verify there are no extraneous bits set beyond the shard committee
//	    for i in range(len(shard_committee), 2 * MAX_PERIOD_COMMITTEE_SIZE):
//	        assert block.aggregation_bits[i] == 0b0
//	    # Verify attester aggregate signature
//	    domain = get_domain(state, DOMAIN_SHARD_ATTESTER, compute_epoch_of_shard_slot(block.slot))
//	    message = hash_tree_root(ShardCheckpoint(shard_state.slot, block.parent_root))
//	    assert bls_verify(bls_aggregate_pubkeys(pubkeys), message, block.attestations, domain)
//	    # Proposer micro-reward
//	    proposer_index = get_shard_proposer_index(state, state.shard, block.slot)
//	    reward = attestation_count * get_base_reward(state, proposer_index) // PROPOSER_REWARD_QUOTIENT
//	    process_delta(state, shard_state, proposer_index, reward)
func ProcessShardAttestations(beaconState *pb.BeaconState, shardState *ethpb.ShardState, shardBlock *ethpb.ShardBlock) (*ethpb.ShardState, error) {
	shardCommittee, err := shardHelper.ShardCommittee(beaconState, shardState.Shard, shardBlock.Slot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get shard commitee")
	}
	pubKeys := make([][]byte, 0, len(shardCommittee))
	attCount := uint64(0)
	for i, validatorIdx := range shardCommittee {
		if shardBlock.AggregationBits[i] == 1 {
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
	end := 2 * params.ShardConfig().MaxPeriodCommitteeSize
	for i := start; i < end; i++ {
		if shardBlock.AggregationBits.BitAt(i) {
			return nil, fmt.Errorf("aggregation bit at index %d should not have been set", i)
		}
	}

	// Verify attester's aggregate signature

	// Handle proposer micro reward
	proposerIdx, err := shardHelper.ShardProposerIndex(beaconState, shardState.Shard, shardBlock.Slot)
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
//  def process_shard_block_body(state: BeaconState, shard_state: ShardState, block: ShardBlock) -> None:
//    # Verify block body size is a multiple of the header size
//    assert len(block.body) % SHARD_HEADER_SIZE == 0
//    # Apply proposer block body fee
//    proposer_index = get_shard_proposer_index(state, state.shard, block.slot)
//    block_body_fee = state.block_body_price * len(block.body) // MAX_SHARD_BLOCK_SIZE
//    process_delta(state, shard_state, proposer_index, -block_body_fee)  # Burn
//    process_delta(state, shard_state, proposer_index, block_body_fee // PROPOSER_REWARD_QUOTIENT)  # Reward
//    # Calculate new block body price
//    block_size = SHARD_HEADER_SIZE + len(block.body)
//    QUOTIENT = MAX_SHARD_BLOCK_SIZE * BLOCK_BODY_PRICE_QUOTIENT
//    price_delta = GweiDelta(state.block_body_price * (block_size - SHARD_BLOCK_SIZE_TARGET) // QUOTIENT)
//    if price_delta > 0:
//        # The maximum block body price caps the amount burnt on fees within a period
//        MAX_BLOCK_BODY_PRICE = MAX_EFFECTIVE_BALANCE // EPOCHS_PER_SHARD_PERIOD // SHARD_SLOTS_PER_EPOCH
//        state.block_body_price = Gwei(min(MAX_BLOCK_BODY_PRICE, state.block_body_price + price_delta))
//    else:
//        state.block_body_price = Gwei(max(MIN_BLOCK_BODY_PRICE, state.block_body_price + price_delta))
func ProcessShardBlockSizeFee(beaconState *pb.BeaconState, shardState *ethpb.ShardState, shardBlock *ethpb.ShardBlock) (*ethpb.ShardState, error) {
	// Charge proposer block size fee
	proposerIdx, err := shardHelper.ShardProposerIndex(beaconState, shardState.Shard, shardBlock.Slot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get proposer index")
	}
	blockSize := uint64(len(shardBlock.Body)) + params.ShardConfig().ShardHeaderSize
	fee := shardState.BlockSizePrice * blockSize / params.ShardConfig().MaxShardBlockSize
	shardState, err = shardHelper.AddFee(beaconState, shardState, proposerIdx, fee)
	if err != nil {
		return nil, errors.Wrap(err, "could not add fee to proposer")
	}

	// Calculate and change new block size pricing
	if blockSize > params.ShardConfig().ShardBlockSizeTarget {
		sizeDelta := blockSize - params.ShardConfig().ShardBlockSizeTarget
		priceDelta := shardState.BlockSizePrice * sizeDelta / params.ShardConfig().MaxShardBlockSize / params.ShardConfig().BlockBodyPriceQuotient
		// Max gas price caps the amount burnt on gas fees within a period to 32ETH
		maxBlockSizePrice := params.BeaconConfig().MaxEffectiveBalance / params.ShardConfig().EpochsPerShardPeriod / params.ShardConfig().ShardSlotsPerEpoch
		if shardState.BlockSizePrice+priceDelta < maxBlockSizePrice {
			shardState.BlockSizePrice = shardState.BlockSizePrice + priceDelta
		} else {
			shardState.BlockSizePrice = maxBlockSizePrice
		}
		return shardState, nil
	}

	sizeDelta := params.ShardConfig().ShardBlockSizeTarget - blockSize
	priceDelta := shardState.BlockSizePrice * sizeDelta / params.ShardConfig().MaxShardBlockSize / params.ShardConfig().BlockBodyPriceQuotient
	if shardState.BlockSizePrice-priceDelta > params.ShardConfig().MinBlockSizePrice {
		shardState.BlockSizePrice = shardState.BlockSizePrice - priceDelta
	} else {
		shardState.BlockSizePrice = params.ShardConfig().MinBlockSizePrice
	}

	return shardState, nil
}
