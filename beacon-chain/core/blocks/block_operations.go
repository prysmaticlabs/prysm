package blocks

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/slices"
	"github.com/prysmaticlabs/prysm/shared/ssz"
	"github.com/prysmaticlabs/prysm/shared/trie"
)

// ProcessPOWReceiptRoots processes the proof-of-work chain's receipts
// contained in a beacon block and appends them as candidate receipt roots
// in the beacon state.
//
// Official spec definition for processing pow receipt roots:
//   If block.candidate_pow_receipt_root is x.candidate_pow_receipt_root
//     for some x in state.candidate_pow_receipt_roots, set x.vote_count += 1.
//   Otherwise, append to state.candidate_pow_receipt_roots a
//   new CandidatePoWReceiptRootRecord(
//     candidate_pow_receipt_root=block.candidate_pow_receipt_root,
//     vote_count=1
//   )
func ProcessPOWReceiptRoots(
	beaconState *pb.BeaconState,
	block *pb.BeaconBlock,
) *pb.BeaconState {
	var newCandidateReceiptRoots []*pb.CandidatePoWReceiptRootRecord
	currentCandidateReceiptRoots := beaconState.CandidatePowReceiptRoots
	for idx, root := range currentCandidateReceiptRoots {
		if bytes.Equal(block.CandidatePowReceiptRootHash32, root.CandidatePowReceiptRootHash32) {
			currentCandidateReceiptRoots[idx].VoteCount++
		} else {
			newCandidateReceiptRoots = append(newCandidateReceiptRoots, &pb.CandidatePoWReceiptRootRecord{
				CandidatePowReceiptRootHash32: block.CandidatePowReceiptRootHash32,
				VoteCount:                     1,
			})
		}
	}
	beaconState.CandidatePowReceiptRoots = append(currentCandidateReceiptRoots, newCandidateReceiptRoots...)
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
	proposerIndex, err := v.BeaconProposerIndex(beaconState, beaconState.Slot)
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
	var latestMix [32]byte
	latestMixesLength := params.BeaconConfig().LatestRandaoMixesLength
	latestMixSlice := beaconState.LatestRandaoMixesHash32S[beaconState.Slot%latestMixesLength]
	copy(latestMix[:], latestMixSlice)
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

func verifyBlockRandao(proposer *pb.ValidatorRecord, block *pb.BeaconBlock) error {
	var blockRandaoReveal [32]byte
	var proposerRandaoCommit [32]byte
	copy(blockRandaoReveal[:], block.RandaoRevealHash32)
	copy(proposerRandaoCommit[:], proposer.RandaoCommitmentHash32)

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
	for idx, slashing := range body.GetProposerSlashings() {
		if err = verifyProposerSlashing(slashing); err != nil {
			return nil, fmt.Errorf("could not verify proposer slashing #%d: %v", idx, err)
		}
		proposer := registry[slashing.GetProposerIndex()]
		if proposer.GetPenalizedSlot() > beaconState.Slot {
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
) error {
	// TODO(#258): Verify BLS according to the specification in the "Proposer Slashings"
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
	return nil
}

// ProcessCasperSlashings is one of the operations performed
// on each processed beacon block to penalize validators based on
// Casper FFG slashing conditions if any slashable events occurred.
//
// Official spec definition for casper slashings:
//
//   Verify that len(block.body.casper_slashings) <= MAX_CASPER_SLASHINGS.
//   For each casper_slashing in block.body.casper_slashings:
//
//   Verify that verify_casper_votes(state, casper_slashing.votes_1).
//   Verify that verify_casper_votes(state, casper_slashing.votes_2).
//   Verify that casper_slashing.votes_1.data != casper_slashing.votes_2.data.
//   Let indices(vote) = vote.aggregate_signature_poc_0_indices +
//     vote.aggregate_signature_poc_1_indices.
//   Let intersection = [x for x in indices(casper_slashing.votes_1)
//     if x in indices(casper_slashing.votes_2)].
//   Verify that len(intersection) >= 1.
//	 Verify the following about the casper votes:
//     (vote1.justified_slot < vote2.justified_slot) &&
//     (vote2.justified_slot + 1 == vote2.slot) &&
//     (vote2.slot < vote1.slot)
//     OR
//     vote1.slot == vote.slot
//   Verify that casper_slashing.votes_1.data.justified_slot + 1 <
//     casper_slashing.votes_2.data.justified_slot + 1 ==
//     casper_slashing.votes_2.data.slot < casper_slashing.votes_1.data.slot
//     or casper_slashing.votes_1.data.slot == casper_slashing.votes_2.data.slot.
//   For each validator index i in intersection,
//     if state.validator_registry[i].penalized_slot > state.slot, then
// 	   run penalize_validator(state, i)
func ProcessCasperSlashings(
	beaconState *pb.BeaconState,
	block *pb.BeaconBlock,
) (*pb.BeaconState, error) {
	body := block.Body
	registry := beaconState.ValidatorRegistry
	if uint64(len(body.CasperSlashings)) > params.BeaconConfig().MaxCasperSlashings {
		return nil, fmt.Errorf(
			"number of casper slashings (%d) exceeds allowed threshold of %d",
			len(body.CasperSlashings),
			params.BeaconConfig().MaxCasperSlashings,
		)
	}
	for idx, slashing := range body.CasperSlashings {
		if err := verifyCasperSlashing(slashing); err != nil {
			return nil, fmt.Errorf("could not verify casper slashing #%d: %v", idx, err)
		}
		validatorIndices, err := casperSlashingPenalizedIndices(slashing)
		if err != nil {
			return nil, fmt.Errorf("could not determine validator indices to penalize: %v", err)
		}
		for _, validatorIndex := range validatorIndices {
			penalizedValidator := registry[validatorIndex]
			if penalizedValidator.GetPenalizedSlot() > beaconState.Slot {
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

func verifyCasperSlashing(slashing *pb.CasperSlashing) error {
	votes1 := slashing.Votes_1
	votes2 := slashing.Votes_2
	votes1Attestation := votes1.Data
	votes2Attestation := votes2.Data

	if err := verifyCasperVotes(votes1); err != nil {
		return fmt.Errorf("could not verify casper votes 1: %v", err)
	}
	if err := verifyCasperVotes(votes2); err != nil {
		return fmt.Errorf("could not verify casper votes 2: %v", err)
	}

	// Inner attestation data structures for the votes should not be equal,
	// as that would mean both votes are the same and therefore no slashing
	// should occur.
	if reflect.DeepEqual(votes1Attestation, votes2Attestation) {
		return fmt.Errorf(
			"casper slashing inner vote attestation data should not match: %v, %v",
			votes1Attestation,
			votes2Attestation,
		)
	}

	// Unless the following holds, the slashing is invalid:
	// (vote1.justified_slot < vote2.justified_slot) &&
	// (vote2.justified_slot + 1 == vote2.slot) &&
	// (vote2.slot < vote1.slot)
	// OR
	// vote1.slot == vote.slot

	justificationValidity := (votes1Attestation.JustifiedSlot < votes2Attestation.JustifiedSlot) &&
		(votes2Attestation.JustifiedSlot+1 == votes2Attestation.Slot) &&
		(votes2Attestation.Slot < votes1Attestation.Slot)

	slotsEqual := votes1Attestation.Slot == votes2Attestation.Slot

	if !(justificationValidity || slotsEqual) {
		return fmt.Errorf(
			`
			Expected the following conditions to hold:
			(vote1.JustifiedSlot < vote2.JustifiedSlot) &&
			(vote2.JustifiedSlot + 1 == vote2.Slot) &&
			(vote2.Slot < vote1.Slot)
			OR
			vote1.Slot == vote.Slot

			Instead, received vote1.JustifiedSlot %d, vote2.JustifiedSlot %d
			and vote1.Slot %d, vote2.Slot %d
			`,
			votes1Attestation.JustifiedSlot,
			votes2Attestation.JustifiedSlot,
			votes1Attestation.Slot,
			votes2Attestation.Slot,
		)
	}
	return nil
}

func casperSlashingPenalizedIndices(slashing *pb.CasperSlashing) ([]uint32, error) {
	votes1 := slashing.Votes_1
	votes2 := slashing.Votes_2
	votes1Indices := append(
		votes1.AggregateSignaturePoc_0Indices,
		votes1.AggregateSignaturePoc_1Indices...,
	)
	votes2Indices := append(
		votes2.AggregateSignaturePoc_0Indices,
		votes2.AggregateSignaturePoc_1Indices...,
	)
	indicesIntersection := slices.Intersection(votes1Indices, votes2Indices)
	if len(indicesIntersection) < 1 {
		return nil, fmt.Errorf(
			"expected intersection of vote indices to be non-empty: %v",
			indicesIntersection,
		)
	}
	return indicesIntersection, nil
}

func verifyCasperVotes(votes *pb.SlashableVoteData) error {
	totalProofsOfCustody := len(votes.AggregateSignaturePoc_0Indices) +
		len(votes.AggregateSignaturePoc_1Indices)
	if uint64(totalProofsOfCustody) > params.BeaconConfig().MaxCasperVotes {
		return fmt.Errorf(
			"exceeded allowed casper votes (%d), received %d",
			params.BeaconConfig().MaxCasperVotes,
			totalProofsOfCustody,
		)
	}
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
		if err := verifyAttestation(beaconState, attestation); err != nil {
			return nil, fmt.Errorf("could not verify attestation at index %d in block: %v", idx, err)
		}
		pendingAttestations = append(pendingAttestations, &pb.PendingAttestationRecord{
			Data:                  attestation.Data,
			ParticipationBitfield: attestation.ParticipationBitfield,
			CustodyBitfield:       attestation.CustodyBitfield,
			SlotIncluded:          beaconState.Slot,
		})
	}
	beaconState.LatestAttestations = pendingAttestations
	return beaconState, nil
}

func verifyAttestation(beaconState *pb.BeaconState, att *pb.Attestation) error {
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
	// Verify that attestation.JustifiedSlot is equal to
	// state.JustifiedSlot if attestation.Slot >=
	// state.Slot - (state.Slot % EPOCH_LENGTH) else state.PreviousJustifiedSlot.
	if att.Data.Slot >= beaconState.Slot-(beaconState.Slot%params.BeaconConfig().EpochLength) {
		if att.Data.JustifiedSlot != beaconState.JustifiedSlot {
			return fmt.Errorf(
				"expected attestation.JustifiedSlot == state.JustifiedSlot, received %d == %d",
				att.Data.JustifiedSlot,
				beaconState.JustifiedSlot,
			)
		}
	} else {
		if att.Data.JustifiedSlot != beaconState.PreviousJustifiedSlot {
			return fmt.Errorf(
				"expected attestation.JustifiedSlot == state.PreviousJustifiedSlot, received %d == %d",
				att.Data.JustifiedSlot,
				beaconState.PreviousJustifiedSlot,
			)
		}
	}

	// Verify that attestation.data.justified_block_root is equal to
	// get_block_root(state, attestation.data.justified_slot).
	blockRoot, err := BlockRoot(beaconState, att.Data.JustifiedSlot)
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
//     depth=DEPOSIT_CONTRACT_TREE_DEPTH and root=state.processed_pow_receipt_root:
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
		// depositData consists of depositInput []byte + depositValue [8]byte +
		// depositTimestamp [8]byte.
		depositValue := depositData[len(depositData)-16 : len(depositData)-8]
		// We then mutate the beacon state with the verified validator deposit.
		beaconState, err = v.ProcessDeposit(
			beaconState,
			validatorIndexMap,
			depositInput.Pubkey,
			binary.BigEndian.Uint64(depositValue),
			depositInput.ProofOfPossession,
			depositInput.WithdrawalCredentialsHash32,
			depositInput.RandaoCommitmentHash32,
			depositInput.PocCommitment,
		)
		if err != nil {
			return nil, fmt.Errorf("could not process deposit into beacon state: %v", err)
		}
	}
	return beaconState, nil
}

// DecodeDepositInput unmarshales a depositData byte slice into
// a proto *pb.DepositInput by using the Simple Serialize (SSZ)
// algorithm.
// TODO(#1253): Do not assume we will receive serialized proto objects - instead,
// replace completely by a common struct which can be simple serialized.
func DecodeDepositInput(depositData []byte) (*pb.DepositInput, error) {
	// Last 16 bytes of deposit data are 8 bytes for value
	// and 8 bytes for timestamp. Everything before that is a
	// Simple Serialized deposit input value.
	if len(depositData) < 16 {
		return nil, fmt.Errorf(
			"deposit data slice too small: len(depositData) = %d",
			len(depositData),
		)
	}
	depositInput := new(pb.DepositInput)
	depositInputBytes := depositData[:len(depositData)-16]
	rBuf := bytes.NewReader(depositInputBytes)
	if err := ssz.Decode(rBuf, depositInput); err != nil {
		return nil, fmt.Errorf("ssz decode failed: %v", err)
	}
	return depositInput, nil
}

func verifyDeposit(beaconState *pb.BeaconState, deposit *pb.Deposit) error {
	depositData := deposit.DepositData
	// Verify Merkle proof of deposit and PoW receipt trie root.
	var receiptRoot [32]byte
	var merkleLeaf [32]byte
	copy(receiptRoot[:], beaconState.ProcessedPowReceiptRootHash32)
	copy(merkleLeaf[:], depositData)
	if ok := trie.VerifyMerkleBranch(
		merkleLeaf,
		deposit.MerkleBranchHash32S,
		params.BeaconConfig().DepositContractTreeDepth,
		receiptRoot,
	); !ok {
		return fmt.Errorf(
			"deposit merkle branch of PoW receipt root did not verify for root: %#x",
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
//     Verify that bls_verify(
//       pubkey=validator.pubkey,
//       message=ZERO_HASH,
//       signature=exit.signature,
//       domain=get_domain(state.fork_data, exit.slot, DOMAIN_EXIT),
//     )
//     Run initiate_validator_exit(
//       state, exit.validator_index,
//     )
func ProcessValidatorExits(
	beaconState *pb.BeaconState,
	block *pb.BeaconBlock,
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
		if err := verifyExit(beaconState, exit); err != nil {
			return nil, fmt.Errorf("could not verify exit #%d: %v", idx, err)
		}
		beaconState = v.InitiateValidatorExit(beaconState, exit.GetValidatorIndex())
	}
	beaconState.ValidatorRegistry = validatorRegistry
	return beaconState, nil
}

func verifyExit(beaconState *pb.BeaconState, exit *pb.Exit) error {
	validator := beaconState.GetValidatorRegistry()[exit.GetValidatorIndex()]
	if validator.GetExitSlot() <= beaconState.Slot+params.BeaconConfig().EntryExitDelay {
		return fmt.Errorf(
			"expected exit.Slot > state.Slot + EntryExitDelay, received %d < %d",
			validator.GetExitSlot(), beaconState.Slot+params.BeaconConfig().EntryExitDelay,
		)
	}
	if beaconState.Slot < exit.Slot {
		return fmt.Errorf(
			"expected state.Slot >= exit.Slot, received %d < %d",
			beaconState.Slot,
			exit.Slot,
		)
	}
	// TODO(#258): Verify using BLS signature verification below:
	// Verify that bls_verify(
	//   pubkey=validator.pubkey,
	//   message=ZERO_HASH,
	//   signature=exit.signature,
	//   domain=get_domain(state.fork_data, exit.slot, DOMAIN_EXIT),
	// )
	return nil
}
