package blocks

import (
	"bytes"
	"fmt"
	"reflect"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/types"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/slices"
)

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
//   Verify that proposer.status != EXITED_WITH_PENALTY.
//   Run update_validator_status(state, proposer_slashing.proposer_index, new_status=EXITED_WITH_PENALTY).
func ProcessProposerSlashings(
	validatorRegistry []*pb.ValidatorRecord,
	proposerSlashings []*pb.ProposerSlashing,
	currentSlot uint64,
) ([]*pb.ValidatorRecord, error) {
	if uint64(len(proposerSlashings)) > params.BeaconConfig().MaxProposerSlashings {
		return nil, fmt.Errorf(
			"number of proposer slashings (%d) exceeds allowed threshold of %d",
			len(proposerSlashings),
			params.BeaconConfig().MaxProposerSlashings,
		)
	}
	for idx, slashing := range proposerSlashings {
		if err := verifyProposerSlashing(slashing); err != nil {
			return nil, fmt.Errorf("could not verify proposer slashing #%d: %v", idx, err)
		}
		proposer := validatorRegistry[slashing.GetProposerIndex()]
		if proposer.Status != pb.ValidatorRecord_EXITED_WITH_PENALTY {
			// TODO(#781): Replace with
			// update_validator_status(
			//   state,
			//   proposer_slashing.proposer_index,
			//   new_status=EXITED_WITH_PENALTY,
			// ) after update_validator_status is implemented.
			validatorRegistry[slashing.GetProposerIndex()] = v.ExitValidator(
				proposer,
				currentSlot,
				true, /* penalize */
			)
		}
	}
	return validatorRegistry, nil
}

func verifyProposerSlashing(
	slashing *pb.ProposerSlashing,
) error {
	// TODO(#781): Verify BLS according to the specification in the "Proposer Slashings"
	// section of block operations.
	slot1 := slashing.GetProposalData_1().GetSlot()
	slot2 := slashing.GetProposalData_2().GetSlot()
	shard1 := slashing.GetProposalData_1().GetShard()
	shard2 := slashing.GetProposalData_2().GetShard()
	root1 := slashing.GetProposalData_1().GetBlockRoot()
	root2 := slashing.GetProposalData_2().GetBlockRoot()
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
//     if state.validator_registry[i].status does not equal
//     EXITED_WITH_PENALTY, then run
//     update_validator_status(state, i, new_status=EXITED_WITH_PENALTY)
func ProcessCasperSlashings(
	validatorRegistry []*pb.ValidatorRecord,
	casperSlashings []*pb.CasperSlashing,
	currentSlot uint64,
) ([]*pb.ValidatorRecord, error) {
	if uint64(len(casperSlashings)) > params.BeaconConfig().MaxCasperSlashings {
		return nil, fmt.Errorf(
			"number of casper slashings (%d) exceeds allowed threshold of %d",
			len(casperSlashings),
			params.BeaconConfig().MaxCasperSlashings,
		)
	}
	for idx, slashing := range casperSlashings {
		if err := verifyCasperSlashing(slashing); err != nil {
			return nil, fmt.Errorf("could not verify casper slashing #%d: %v", idx, err)
		}
		validatorIndices, err := casperSlashingPenalizedIndices(slashing)
		if err != nil {
			return nil, fmt.Errorf("could not determine validator indices to penalize: %v", err)
		}
		for _, validatorIndex := range validatorIndices {
			penalizedValidator := validatorRegistry[validatorIndex]
			if penalizedValidator.Status != pb.ValidatorRecord_EXITED_WITH_PENALTY {
				// TODO(#781): Replace with update_validator_status(
				//   state,
				//   validatorIndex,
				//   new_status=EXITED_WITH_PENALTY,
				// ) after update_validator_status is implemented.
				validatorRegistry[validatorIndex] = v.ExitValidator(
					penalizedValidator,
					currentSlot,
					true, /* penalize */
				)
			}
		}
	}
	return validatorRegistry, nil
}

func verifyCasperSlashing(slashing *pb.CasperSlashing) error {
	votes1 := slashing.GetVotes_1()
	votes2 := slashing.GetVotes_2()
	votes1Attestation := votes1.GetData()
	votes2Attestation := votes2.GetData()

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

	justificationValidity := (votes1Attestation.GetJustifiedSlot() < votes2Attestation.GetJustifiedSlot()) &&
		(votes2Attestation.GetJustifiedSlot()+1 == votes2Attestation.GetSlot()) &&
		(votes2Attestation.GetSlot() < votes1Attestation.GetSlot())

	slotsEqual := votes1Attestation.GetSlot() == votes2Attestation.GetSlot()

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
			votes1Attestation.GetJustifiedSlot(),
			votes2Attestation.GetJustifiedSlot(),
			votes1Attestation.GetSlot(),
			votes2Attestation.GetSlot(),
		)
	}
	return nil
}

func casperSlashingPenalizedIndices(slashing *pb.CasperSlashing) ([]uint32, error) {
	votes1 := slashing.GetVotes_1()
	votes2 := slashing.GetVotes_2()
	votes1Indices := append(
		votes1.GetAggregateSignaturePoc_0Indices(),
		votes1.GetAggregateSignaturePoc_1Indices()...,
	)
	votes2Indices := append(
		votes2.GetAggregateSignaturePoc_0Indices(),
		votes2.GetAggregateSignaturePoc_1Indices()...,
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
	totalProofsOfCustody := len(votes.GetAggregateSignaturePoc_0Indices()) +
		len(votes.GetAggregateSignaturePoc_1Indices())
	if uint64(totalProofsOfCustody) > params.BeaconConfig().MaxCasperVotes {
		return fmt.Errorf(
			"exceeded allowed casper votes (%d), received %d",
			params.BeaconConfig().MaxCasperVotes,
			totalProofsOfCustody,
		)
	}
	// TODO(#781): Implement BLS verify multiple.
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
	beaconState *types.BeaconState,
	block *pb.BeaconBlock,
) ([]*pb.PendingAttestationRecord, error) {
	atts := block.GetBody().GetAttestations()
	if uint64(len(atts)) > params.BeaconConfig().MaxAttestations {
		return nil, fmt.Errorf(
			"number of attestations in block  (%d) exceeds allowed threshold of %d",
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
			Data:                  attestation.GetData(),
			ParticipationBitfield: attestation.GetParticipationBitfield(),
			CustodyBitfield:       attestation.GetCustodyBitfield(),
			SlotIncluded:          beaconState.Slot(),
		})
	}
	return pendingAttestations, nil
}

func verifyAttestation(beaconState *types.BeaconState, att *pb.Attestation) error {
	inclusionDelay := params.BeaconConfig().MinAttestationInclusionDelay * params.BeaconConfig().SlotDuration
	if att.GetData().GetSlot()+inclusionDelay > beaconState.Slot() {
		return fmt.Errorf(
			"attestation slot (slot %d) + inclusion delay (%d) beyond current beacon state slot (%d)",
			att.GetData().GetSlot(),
			inclusionDelay,
			beaconState.Slot(),
		)
	}
	if att.GetData().GetSlot()+params.BeaconConfig().EpochLength < beaconState.Slot() {
		return fmt.Errorf(
			"attestation slot (slot %d) + epoch length (%d) less than current beacon state slot (%d)",
			att.GetData().GetSlot(),
			params.BeaconConfig().EpochLength,
			beaconState.Slot(),
		)
	}
	//   Verify that either attestation.data.latest_crosslink_root or
	//     attestation.data.shard_block_root equals
	//     state.latest_crosslinks[shard].shard_block_root
	crossLinkRoot := att.GetData().GetLatestCrosslinkRoot()
	shardBlockRoot := att.GetData().GetShardBlockRoot()
	stateShardBlockRoot := beaconState.LatestCrosslinks()
	fmt.Printf("%v%v%v", crossLinkRoot, shardBlockRoot, stateShardBlockRoot)

	// Verify attestation.shard_block_root == ZERO_HASH [TO BE REMOVED IN PHASE 1].
	if !bytes.Equal(att.GetData().GetShardBlockRoot(), []byte{}) {
		return fmt.Errorf(
			"expected attestation's shard block root == %#x, received %#x instead",
			[]byte{},
			att.GetData().GetShardBlockRoot(),
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
