// Package blocks contains block processing libraries. These libraries
// process and verify block specific messages such as PoW receipt root,
// RANDAO, validator deposits, exits and slashing proofs.
package blocks

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

// VerifyProposerSignature uses BLS signature verification to ensure
// the correct proposer created an incoming beacon block during state
// transition processing.
//
// WIP - this is stubbed out until BLS is integrated into Prysm.
func VerifyProposerSignature(
	block *pb.BeaconBlock,
) error {
	if block == nil {
		return errors.New("received nil block")
	}
	return nil
}

// ProcessEth1Data is an operation performed on each
// beacon block to ensure the ETH1 data votes are processed
// into the beacon state.
//
// Official spec definition of ProcessEth1Data
//   If block.eth1_data equals eth1_data_vote.eth1_data for some eth1_data_vote
//   in state.eth1_data_votes, set eth1_data_vote.vote_count += 1.
//   Otherwise, append to state.eth1_data_votes a new Eth1DataVote(eth1_data=block.eth1_data, vote_count=1).
func ProcessEth1Data(beaconState *pb.BeaconState, block *pb.BeaconBlock) *pb.BeaconState {
	var eth1DataVoteAdded bool

	for _, Eth1DataVote := range beaconState.Eth1DataVotes {
		if bytes.Equal(Eth1DataVote.Eth1Data.BlockHash32, block.Eth1Data.BlockHash32) && bytes.Equal(Eth1DataVote.Eth1Data.DepositRootHash32, block.Eth1Data.DepositRootHash32) {
			Eth1DataVote.VoteCount++
			eth1DataVoteAdded = true
			break
		}
	}

	if !eth1DataVoteAdded {
		beaconState.Eth1DataVotes = append(
			beaconState.Eth1DataVotes,
			&pb.Eth1DataVote{
				Eth1Data:  block.Eth1Data,
				VoteCount: 1,
			},
		)
	}

	return beaconState
}

// ProcessBlockRandao checks the block proposer's
// randao commitment and generates a new randao mix to update
// in the beacon state's latest randao mixes and set the proposer's randao fields.
//
// Official spec definition for block randao verification:
//   Let repeat_hash(x, n) = x if n == 0 else repeat_hash(hash(x), n-1).
//   Let proposer = state.validator_registry[get_beacon_proposer_index(state, state.slot)].
//   Verify that repeat_hash(block.randao_reveal, proposer.randao_layers) == proposer.randao_commitment.
//   Set state.latest_randao_mixes[state.slot % LATEST_RANDAO_MIXES_LENGTH] =
//     xor(state.latest_randao_mixes[state.slot % LATEST_RANDAO_MIXES_LENGTH], block.randao_reveal)
//   Set proposer.randao_commitment = block.randao_reveal.
//   Set proposer.randao_layers = 0
func ProcessBlockRandao(beaconState *pb.BeaconState, block *pb.BeaconBlock) (*pb.BeaconState, error) {
	proposerIndex, err := v.BeaconProposerIdx(beaconState, beaconState.Slot)
	if err != nil {
		return nil, fmt.Errorf("could not fetch beacon proposer index: %v", err)
	}
	registry := beaconState.ValidatorRegistry
	proposer := registry[proposerIndex]
	if err := verifyBlockRandao(proposer, block); err != nil {
		return nil, fmt.Errorf("could not verify block randao: %v", err)
	}
	// If block randao passed verification, we XOR the state's latest randao mix with the block's
	// randao and update the state's corresponding latest randao mix value.
	latestMixesLength := params.BeaconConfig().LatestRandaoMixesLength
	latestMixSlice := beaconState.LatestRandaoMixesHash32S[beaconState.Slot%latestMixesLength]
	latestMix := bytesutil.ToBytes32(latestMixSlice)
	for i, x := range block.RandaoRevealHash32 {
		latestMix[i] ^= x
	}
	proposer.RandaoCommitmentHash32 = block.RandaoRevealHash32
	proposer.RandaoLayers = 0
	registry[proposerIndex] = proposer
	beaconState.LatestRandaoMixesHash32S[beaconState.Slot%latestMixesLength] = latestMix[:]
	beaconState.ValidatorRegistry = registry
	return beaconState, nil
}

func verifyBlockRandao(proposer *pb.Validator, block *pb.BeaconBlock) error {
	blockRandaoReveal := bytesutil.ToBytes32(block.RandaoRevealHash32)
	proposerRandaoCommit := bytesutil.ToBytes32(proposer.RandaoCommitmentHash32)
	randaoHashLayers := hashutil.RepeatHash(blockRandaoReveal, proposer.RandaoLayers)
	// Verify that repeat_hash(block.randao_reveal, proposer.randao_layers) == proposer.randao_commitment.
	if randaoHashLayers != proposerRandaoCommit {
		return fmt.Errorf(
			"expected hashed block randao layers to equal proposer randao: received %#x = %#x",
			randaoHashLayers[:],
			proposerRandaoCommit[:],
		)
	}
	return nil
}

// ProcessProposerSlashings is one of the operations performed
// on each processed beacon block to penalize proposers based on
// slashing conditions if any slashable events occurred.
//
// Official spec definition for proposer slashings:
//   Verify that len(block.body.proposer_slashings) <= MAX_PROPOSER_SLASHINGS.
//
//   For each proposer_slashing in block.body.proposer_slashings:
//
//   Let proposer = state.validator_registry[proposer_slashing.proposer_index].
//   Verify that bls_verify(pubkey=proposer.pubkey, msg=hash_tree_root(
//    proposer_slashing.proposal_data_1),
//  	sig=proposer_slashing.proposal_signature_1,
//    domain=get_domain(state.fork_data, proposer_slashing.proposal_data_1.slot, DOMAIN_PROPOSAL)).
//   Verify that bls_verify(pubkey=proposer.pubkey, msg=hash_tree_root(
//     proposer_slashing.proposal_data_2),
//     sig=proposer_slashing.proposal_signature_2,
//     domain=get_domain(state.fork_data, proposer_slashing.proposal_data_2.slot, DOMAIN_PROPOSAL)).
//   Verify that proposer_slashing.proposal_data_1.slot == proposer_slashing.proposal_data_2.slot.
//   Verify that proposer_slashing.proposal_data_1.shard == proposer_slashing.proposal_data_2.shard.
//   Verify that proposer_slashing.proposal_data_1.block_root != proposer_slashing.proposal_data_2.block_root.
//   Verify that validator.penalized_slot > state.slot.
//   Run penalize_validator(state, proposer_slashing.proposer_index).
func ProcessProposerSlashings(
	beaconState *pb.BeaconState,
	block *pb.BeaconBlock,
	verifySignatures bool,
) (*pb.BeaconState, error) {
	body := block.Body
	registry := beaconState.ValidatorRegistry
	if uint64(len(body.ProposerSlashings)) > params.BeaconConfig().MaxProposerSlashings {
		return nil, fmt.Errorf(
			"number of proposer slashings (%d) exceeds allowed threshold of %d",
			len(body.ProposerSlashings),
			params.BeaconConfig().MaxProposerSlashings,
		)
	}
	var err error
	for idx, slashing := range body.ProposerSlashings {
		if err = verifyProposerSlashing(slashing, verifySignatures); err != nil {
			return nil, fmt.Errorf("could not verify proposer slashing #%d: %v", idx, err)
		}
		proposer := registry[slashing.ProposerIndex]
		if proposer.PenalizedEpoch > beaconState.Slot {
			beaconState, err = v.PenalizeValidator(beaconState, slashing.ProposerIndex)
			if err != nil {
				return nil, fmt.Errorf("could not penalize proposer index %d: %v",
					slashing.ProposerIndex, err)
			}
		}
	}
	return beaconState, nil
}

func verifyProposerSlashing(
	slashing *pb.ProposerSlashing,
	verifySignatures bool,
) error {
	// section of block operations.
	slot1 := slashing.ProposalData_1.Slot
	slot2 := slashing.ProposalData_2.Slot
	shard1 := slashing.ProposalData_1.Shard
	shard2 := slashing.ProposalData_2.Shard
	root1 := slashing.ProposalData_1.BlockRootHash32
	root2 := slashing.ProposalData_2.BlockRootHash32
	if slot1 != slot2 {
		return fmt.Errorf("slashing proposal data slots do not match: %d, %d", slot1, slot2)
	}
	if shard1 != shard2 {
		return fmt.Errorf("slashing proposal data shards do not match: %d, %d", shard1, shard2)
	}
	if !bytes.Equal(root1, root2) {
		return fmt.Errorf("slashing proposal data block roots do not match: %#x, %#x", root1, root2)
	}
	if verifySignatures {
		// TODO(#258): Verify BLS according to the specification in the "Proposer Slashings"
		return nil
	}
	return nil
}

// ProcessAttesterSlashings is one of the operations performed
// on each processed beacon block to penalize validators based on
// Casper FFG slashing conditions if any slashable events occurred.
//
// Official spec definition for attester slashings:
//
//   Verify that len(block.body.attester_slashings) <= MAX_CASPER_SLASHINGS.
//   For each attester_slashing in block.body.attester_slashings:
//
//   Verify that verify_attester_votes(state, attester_slashing.votes_1).
//   Verify that verify_attester_votes(state, attester_slashing.votes_2).
//   Verify that attester_slashing.votes_1.data != attester_slashing.votes_2.data.
//   Let indices(vote) = vote.aggregate_signature_poc_0_indices +
//     vote.aggregate_signature_poc_1_indices.
//   Let intersection = [x for x in indices(attester_slashing.votes_1)
//     if x in indices(attester_slashing.votes_2)].
//   Verify that len(intersection) >= 1.
//	 Verify the following about the attester votes:
//     (vote1.justified_slot < vote2.justified_slot) &&
//     (vote2.justified_slot + 1 == vote2.slot) &&
//     (vote2.slot < vote1.slot)
//     OR
//     vote1.slot == vote.slot
//   Verify that attester_slashing.votes_1.data.justified_slot + 1 <
//     attester_slashing.votes_2.data.justified_slot + 1 ==
//     attester_slashing.votes_2.data.slot < attester_slashing.votes_1.data.slot
//     or attester_slashing.votes_1.data.slot == attester_slashing.votes_2.data.slot.
//   For each validator index i in intersection,
//     if state.validator_registry[i].penalized_slot > state.slot, then
// 	   run penalize_validator(state, i)
func ProcessAttesterSlashings(
	beaconState *pb.BeaconState,
	block *pb.BeaconBlock,
	verifySignatures bool,
) (*pb.BeaconState, error) {
	body := block.Body
	registry := beaconState.ValidatorRegistry
	if uint64(len(body.AttesterSlashings)) > params.BeaconConfig().MaxAttesterSlashings {
		return nil, fmt.Errorf(
			"number of attester slashings (%d) exceeds allowed threshold of %d",
			len(body.AttesterSlashings),
			params.BeaconConfig().MaxAttesterSlashings,
		)
	}
	for idx, slashing := range body.AttesterSlashings {
		if err := verifyAttesterSlashing(slashing, verifySignatures); err != nil {
			return nil, fmt.Errorf("could not verify attester slashing #%d: %v", idx, err)
		}
		validatorIndices, err := attesterSlashingPenalizedIndices(slashing)
		if err != nil {
			return nil, fmt.Errorf("could not determine validator indices to penalize: %v", err)
		}
		for _, validatorIndex := range validatorIndices {
			penalizedValidator := registry[validatorIndex]
			if penalizedValidator.PenalizedEpoch > beaconState.Slot {
				beaconState, err = v.PenalizeValidator(beaconState, validatorIndex)
				if err != nil {
					return nil, fmt.Errorf("could not penalize validator index %d: %v",
						validatorIndex, err)
				}
			}
		}
	}
	return beaconState, nil
}

func verifyAttesterSlashing(slashing *pb.AttesterSlashing, verifySignatures bool) error {
	slashableVote1 := slashing.SlashableVote_1
	slashableVote2 := slashing.SlashableVote_2
	slashableVoteData1Attestation := slashableVote1.Data
	slashableVoteData2Attestation := slashableVote2.Data

	if err := verifySlashableVote(slashableVote1, verifySignatures); err != nil {
		return fmt.Errorf("could not verify attester slashable vote data 1: %v", err)
	}
	if err := verifySlashableVote(slashableVote2, verifySignatures); err != nil {
		return fmt.Errorf("could not verify attester slashable vote data 2: %v", err)
	}

	// Inner attestation data structures for the votes should not be equal,
	// as that would mean both votes are the same and therefore no slashing
	// should occur.
	if reflect.DeepEqual(slashableVoteData1Attestation, slashableVoteData2Attestation) {
		return fmt.Errorf(
			"attester slashing inner slashable vote data attestation should not match: %v, %v",
			slashableVoteData1Attestation,
			slashableVoteData2Attestation,
		)
	}

	// Unless the following holds, the slashing is invalid:
	// (vote1.justified_slot < vote2.justified_slot) &&
	// (vote2.justified_slot + 1 == vote2.slot) &&
	// (vote2.slot < vote1.slot)
	// OR
	// vote1.slot == vote2.slot
	justificationValidity :=
		(slashableVoteData1Attestation.JustifiedSlot < slashableVoteData2Attestation.JustifiedSlot) &&
			(slashableVoteData2Attestation.JustifiedSlot+1 == slashableVoteData2Attestation.Slot) &&
			(slashableVoteData2Attestation.Slot < slashableVoteData1Attestation.Slot)

	slotsEqual := slashableVoteData1Attestation.Slot == slashableVoteData2Attestation.Slot

	if !(justificationValidity || slotsEqual) {
		return fmt.Errorf(
			`
			Expected the following conditions to hold:
			(slashableVoteData1.JustifiedSlot <
			slashableVoteData2.JustifiedSlot) &&
			(slashableVoteData2.JustifiedSlot + 1
			== slashableVoteData1.Slot) &&
			(slashableVoteData2.Slot < slashableVoteData1.Slot)
			OR
			slashableVoteData1.Slot == slashableVoteData2.Slot

			Instead, received slashableVoteData1.JustifiedSlot %d,
			slashableVoteData2.JustifiedSlot %d
			and slashableVoteData1.Slot %d, slashableVoteData2.Slot %d
			`,
			slashableVoteData1Attestation.JustifiedSlot,
			slashableVoteData2Attestation.JustifiedSlot,
			slashableVoteData1Attestation.Slot,
			slashableVoteData2Attestation.Slot,
		)
	}
	return nil
}

func attesterSlashingPenalizedIndices(slashing *pb.AttesterSlashing) ([]uint64, error) {
	indicesIntersection := sliceutil.Intersection(
		slashing.SlashableVote_1.ValidatorIndices,
		slashing.SlashableVote_2.ValidatorIndices)
	if len(indicesIntersection) < 1 {
		return nil, fmt.Errorf(
			"expected intersection of vote indices to be non-empty: %v",
			indicesIntersection,
		)
	}
	return indicesIntersection, nil
}

func verifySlashableVote(votes *pb.SlashableVote, verifySignatures bool) error {
	emptyCustody := make([]byte, len(votes.CustodyBitfield))
	if bytes.Equal(votes.CustodyBitfield, emptyCustody) {
		return errors.New("custody bit field can't all be 0s")
	}
	if len(votes.ValidatorIndices) == 0 {
		return errors.New("empty validator indices")
	}
	for i := 0; i < len(votes.ValidatorIndices)-1; i++ {
		if votes.ValidatorIndices[i] >= votes.ValidatorIndices[i+1] {
			return fmt.Errorf("validator indices not in descending order: %v",
				votes.ValidatorIndices)
		}
	}
	if len(votes.CustodyBitfield) != mathutil.CeilDiv8(len(votes.ValidatorIndices)) {
		return fmt.Errorf("custody bit field length (%d) don't match indices length (%d)",
			len(votes.CustodyBitfield), mathutil.CeilDiv8(len(votes.ValidatorIndices)))
	}
	if uint64(len(votes.ValidatorIndices)) > params.BeaconConfig().MaxIndicesPerSlashableVote {
		return fmt.Errorf("validator indices length (%d) exceeded max indices per slashable vote(%d)",
			len(votes.ValidatorIndices), params.BeaconConfig().MaxIndicesPerSlashableVote)
	}

	if verifySignatures {
		// TODO(#258): Implement BLS verify multiple.
		//  pubs = aggregate_pubkeys for each validator in registry for poc0 and poc1
		//    indices
		//  bls_verify_multiple(
		//    pubkeys=pubs,
		//    messages=[
		//      hash_tree_root(votes)+bytes1(0),
		//      hash_tree_root(votes)+bytes1(1),
		//      signature=aggregate_signature
		//    ]
		//  )
		return nil
	}
	return nil
}

// ProcessBlockAttestations applies processing operations to a block's inner attestation
// records. This function returns a list of pending attestations which can then be
// appended to the BeaconState's latest attestations.
//
// Official spec definition for block attestation processing:
//   Verify that len(block.body.attestations) <= MAX_ATTESTATIONS.
//
//   For each attestation in block.body.attestations:
//   Verify that attestation.data.slot + MIN_ATTESTATION_INCLUSION_DELAY <= state.slot.
//   Verify that attestation.data.slot + EPOCH_LENGTH >= state.slot.
//   Verify that attestation.data.justified_slot is equal to
//     state.justified_slot if attestation.data.slot >=
//     state.slot - (state.slot % EPOCH_LENGTH) else state.previous_justified_slot.
//   Verify that attestation.data.justified_block_root is equal to
//     get_block_root(state, attestation.data.justified_slot).
//   Verify that either attestation.data.latest_crosslink_root or
//     attestation.data.shard_block_root equals
//     state.latest_crosslinks[shard].shard_block_root
//   Aggregate_signature verification:
//     Let participants = get_attestation_participants(
//       state,
//       attestation.data,
//       attestation.participation_bitfield,
//     )
//     Let group_public_key = BLSAddPubkeys([
//       state.validator_registry[v].pubkey for v in participants
//     ])
//     Verify that bls_verify(
//       pubkey=group_public_key,
//       message=hash_tree_root(attestation.data) + bytes1(0),
//       signature=attestation.aggregate_signature,
//       domain=get_domain(state.fork_data, attestation.data.slot, DOMAIN_ATTESTATION)).
//
//   [TO BE REMOVED IN PHASE 1] Verify that attestation.data.shard_block_hash == ZERO_HASH.
//   return PendingAttestationRecord(
//     data=attestation.data,
//     participation_bitfield=attestation.participation_bitfield,
//     custody_bitfield=attestation.custody_bitfield,
//     slot_included=state.slot,
//   ) which can then be appended to state.latest_attestations.
func ProcessBlockAttestations(
	beaconState *pb.BeaconState,
	block *pb.BeaconBlock,
	verifySignatures bool,
) (*pb.BeaconState, error) {
	atts := block.Body.Attestations
	if uint64(len(atts)) > params.BeaconConfig().MaxAttestations {
		return nil, fmt.Errorf(
			"number of attestations in block (%d) exceeds allowed threshold of %d",
			len(atts),
			params.BeaconConfig().MaxAttestations,
		)
	}
	var pendingAttestations []*pb.PendingAttestationRecord
	for idx, attestation := range atts {
		if err := verifyAttestation(beaconState, attestation, verifySignatures); err != nil {
			return nil, fmt.Errorf("could not verify attestation at index %d in block: %v", idx, err)
		}
		pendingAttestations = append(pendingAttestations, &pb.PendingAttestationRecord{
			Data:                attestation.Data,
			AggregationBitfield: attestation.AggregationBitfield,
			CustodyBitfield:     attestation.CustodyBitfield,
			SlotIncluded:        beaconState.Slot,
		})
	}
	beaconState.LatestAttestations = pendingAttestations
	return beaconState, nil
}

func verifyAttestation(beaconState *pb.BeaconState, att *pb.Attestation, verifySignatures bool) error {
	inclusionDelay := params.BeaconConfig().MinAttestationInclusionDelay
	if att.Data.Slot+inclusionDelay > beaconState.Slot {
		return fmt.Errorf(
			"attestation slot (slot %d) + inclusion delay (%d) beyond current beacon state slot (%d)",
			att.Data.Slot,
			inclusionDelay,
			beaconState.Slot,
		)
	}
	if att.Data.Slot+params.BeaconConfig().EpochLength < beaconState.Slot {
		return fmt.Errorf(
			"attestation slot (slot %d) + epoch length (%d) less than current beacon state slot (%d)",
			att.Data.Slot,
			params.BeaconConfig().EpochLength,
			beaconState.Slot,
		)
	}
	// Verify that attestation.data.justified_epoch is equal to state.justified_epoch
	// 	if attestation.data.slot >= get_epoch_start_slot(get_current_epoch(state))
	// 	else state.previous_justified_epoch.
	if att.Data.Slot >= helpers.StartSlot(helpers.SlotToEpoch(beaconState.Slot)) {
		if att.Data.JustifiedEpoch != beaconState.JustifiedEpoch {
			return fmt.Errorf(
				"expected attestation.JustifiedEpoch == state.JustifiedEpoch, received %d == %d",
				att.Data.JustifiedEpoch,
				beaconState.JustifiedEpoch,
			)
		}
	} else {
		if att.Data.JustifiedEpoch != beaconState.PreviousJustifiedEpoch {
			return fmt.Errorf(
				"expected attestation.JustifiedEpoch == state.PreviousJustifiedEpoch, received %d == %d",
				att.Data.JustifiedEpoch,
				beaconState.PreviousJustifiedEpoch,
			)
		}
	}

	// Verify that attestation.data.justified_block_root is equal to
	// get_block_root(state, get_epoch_start_slot(attestation.data.justified_epoch)).
	blockRoot, err := BlockRoot(beaconState, helpers.StartSlot(att.Data.JustifiedEpoch))
	if err != nil {
		return fmt.Errorf("could not get block root for justified slot: %v", err)
	}

	justifiedBlockRoot := att.Data.JustifiedBlockRootHash32
	if !bytes.Equal(justifiedBlockRoot, blockRoot) {
		return fmt.Errorf(
			"expected JustifiedBlockRoot == getBlockRoot(state, JustifiedSlot): got %#x = %#x",
			justifiedBlockRoot,
			blockRoot,
		)
	}

	// Verify that either: attestation.data.latest_crosslink_root or
	// attestation.data.shard_block_root equals
	// state.latest_crosslinks[shard].shard_block_root
	crossLinkRoot := att.Data.LatestCrosslinkRootHash32
	shardBlockRoot := att.Data.ShardBlockRootHash32
	shard := att.Data.Shard
	stateShardBlockRoot := beaconState.LatestCrosslinks[shard].ShardBlockRootHash32

	if !(bytes.Equal(crossLinkRoot, stateShardBlockRoot) ||
		bytes.Equal(shardBlockRoot, stateShardBlockRoot)) {
		return fmt.Errorf(
			"attestation.CrossLinkRoot and ShardBlockRoot != %v (state.LatestCrosslinks' ShardBlockRoot)",
			stateShardBlockRoot,
		)
	}

	// Verify attestation.shard_block_root == ZERO_HASH [TO BE REMOVED IN PHASE 1].
	if !bytes.Equal(att.Data.ShardBlockRootHash32, []byte{}) {
		return fmt.Errorf(
			"expected attestation.ShardBlockRoot == %#x, received %#x instead",
			[]byte{},
			att.Data.ShardBlockRootHash32,
		)
	}
	if verifySignatures {
		// TODO(#258): Integrate BLS signature verification for attestation.
		//     Let participants = get_attestation_participants(
		//       state,
		//       attestation.data,
		//       attestation.participation_bitfield,
		//     )
		//     Let group_public_key = BLSAddPubkeys([
		//       state.validator_registry[v].pubkey for v in participants
		//     ])
		//     Verify that bls_verify(
		//       pubkey=group_public_key,
		//       message=hash_tree_root(attestation.data) + bytes1(0),
		//       signature=attestation.aggregate_signature,
		//       domain=get_domain(state.fork_data, attestation.data.slot, DOMAIN_ATTESTATION)).
		return nil
	}
	return nil
}

// ProcessValidatorDeposits is one of the operations performed on each processed
// beacon block to verify queued validators from the Ethereum 1.0 Deposit Contract
// into the beacon chain.
//
// Official spec definition for processing validator deposits:
//   Verify that len(block.body.deposits) <= MAX_DEPOSITS.
//   For each deposit in block.body.deposits:
//     Let serialized_deposit_data be the serialized form of deposit.deposit_data.
//     It should be the DepositInput followed by 8 bytes for deposit_data.value
//     and 8 bytes for deposit_data.timestamp. That is, it should match
//     deposit_data in the Ethereum 1.0 deposit contract of which the hash
//     was placed into the Merkle tree.
//
//     Verify deposit merkle_branch, setting leaf=serialized_deposit_data,
//     depth=DEPOSIT_CONTRACT_TREE_DEPTH and root=state.latest_deposit_root:
//
//     Run the following:
//     process_deposit(
//       state=state,
//       pubkey=deposit.deposit_data.deposit_input.pubkey,
//       deposit=deposit.deposit_data.value,
//       proof_of_possession=deposit.deposit_data.deposit_input.proof_of_possession,
//       withdrawal_credentials=deposit.deposit_data.deposit_input.withdrawal_credentials,
//       randao_commitment=deposit.deposit_data.deposit_input.randao_commitment,
//       poc_commitment=deposit.deposit_data.deposit_input.poc_commitment,
//     )
func ProcessValidatorDeposits(
	beaconState *pb.BeaconState,
	block *pb.BeaconBlock,
) (*pb.BeaconState, error) {
	deposits := block.Body.Deposits
	if uint64(len(deposits)) > params.BeaconConfig().MaxDeposits {
		return nil, fmt.Errorf(
			"number of deposits (%d) exceeds allowed threshold of %d",
			len(deposits),
			params.BeaconConfig().MaxDeposits,
		)
	}
	var err error
	var depositInput *pb.DepositInput
	validatorIndexMap := stateutils.ValidatorIndexMap(beaconState)
	for idx, deposit := range deposits {
		depositData := deposit.DepositData
		depositInput, err = DecodeDepositInput(depositData)
		if err != nil {
			return nil, fmt.Errorf("could not decode deposit input: %v", err)
		}
		if err = verifyDeposit(beaconState, deposit); err != nil {
			return nil, fmt.Errorf("could not verify deposit #%d: %v", idx, err)
		}
		// depositData consists of depositValue [8]byte +
		// depositTimestamp [8]byte + depositInput []byte .
		depositValue := depositData[:8]
		// We then mutate the beacon state with the verified validator deposit.
		beaconState, err = v.ProcessDeposit(
			beaconState,
			validatorIndexMap,
			depositInput.Pubkey,
			binary.BigEndian.Uint64(depositValue),
			depositInput.ProofOfPossession,
			depositInput.WithdrawalCredentialsHash32,
			depositInput.RandaoCommitmentHash32,
		)
		if err != nil {
			return nil, fmt.Errorf("could not process deposit into beacon state: %v", err)
		}
	}
	return beaconState, nil
}

func verifyDeposit(beaconState *pb.BeaconState, deposit *pb.Deposit) error {
	// Verify Merkle proof of deposit and deposit trie root.
	receiptRoot := bytesutil.ToBytes32(beaconState.LatestEth1Data.DepositRootHash32)
	if ok := trieutil.VerifyMerkleBranch(
		hashutil.Hash(deposit.DepositData),
		deposit.MerkleBranchHash32S,
		params.BeaconConfig().DepositContractTreeDepth,
		deposit.MerkleTreeIndex,
		receiptRoot,
	); !ok {
		return fmt.Errorf(
			"deposit merkle branch of deposit root did not verify for root: %#x",
			receiptRoot,
		)
	}

	return nil
}

// ProcessValidatorExits is one of the operations performed
// on each processed beacon block to determine which validators
// should exit the state's validator registry.
//
// Official spec definition for processing exits:
//
//   Verify that len(block.body.exits) <= MAX_EXITS.
//
//   For each exit in block.body.exits:
//     Let validator = state.validator_registry[exit.validator_index].
//     Verify that validator.exit_slot > state.slot + ENTRY_EXIT_DELAY.
//     Verify that state.slot >= exit.slot.
//     Verify that state.slot >= validator.latest_status_change_slot +
//       SHARD_PERSISTENT_COMMITTEE_CHANGE_PERIOD.
//     Let exit_message = hash_tree_root(
//       Exit(
//         slot=exit.slot,
//         validator_index=exit.validator_index,
//         signature=EMPTY_SIGNATURE
//       )
//     ).
//     Verify that bls_verify(
//       pubkey=validator.pubkey,
//       message=exit_message,
//       signature=exit.signature,
//       domain=get_domain(state.fork_data, exit.slot, DOMAIN_EXIT),
//     )
//     Run initiate_validator_exit(
//       state, exit.validator_index,
//     )
func ProcessValidatorExits(
	beaconState *pb.BeaconState,
	block *pb.BeaconBlock,
	verifySignatures bool,
) (*pb.BeaconState, error) {
	exits := block.Body.Exits
	if uint64(len(exits)) > params.BeaconConfig().MaxExits {
		return nil, fmt.Errorf(
			"number of exits (%d) exceeds allowed threshold of %d",
			len(exits),
			params.BeaconConfig().MaxExits,
		)
	}

	validatorRegistry := beaconState.ValidatorRegistry
	for idx, exit := range exits {
		if err := verifyExit(beaconState, exit, verifySignatures); err != nil {
			return nil, fmt.Errorf("could not verify exit #%d: %v", idx, err)
		}
		beaconState = v.InitiateValidatorExit(beaconState, exit.ValidatorIndex)
	}
	beaconState.ValidatorRegistry = validatorRegistry
	return beaconState, nil
}

func verifyExit(beaconState *pb.BeaconState, exit *pb.Exit, verifySignatures bool) error {
	validator := beaconState.ValidatorRegistry[exit.ValidatorIndex]
	if validator.ExitEpoch <= beaconState.Slot+params.BeaconConfig().EntryExitDelay {
		return fmt.Errorf(
			"expected exit.Slot > state.Slot + EntryExitDelay, received %d < %d",
			validator.ExitEpoch, beaconState.Slot+params.BeaconConfig().EntryExitDelay,
		)
	}
	if beaconState.Slot < exit.Slot {
		return fmt.Errorf(
			"expected state.Slot >= exit.Slot, received %d < %d",
			beaconState.Slot,
			exit.Slot,
		)
	}
	if verifySignatures {
		// TODO(#258): Verify using BLS signature verification below:
		// Verify that bls_verify(
		//   pubkey=validator.pubkey,
		//   message=ZERO_HASH,
		//   signature=exit.signature,
		//   domain=get_domain(state.fork_data, exit.slot, DOMAIN_EXIT),
		// )
		return nil
	}
	return nil
}
