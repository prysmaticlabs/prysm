package blocks

import (
	"bytes"
	"fmt"
	"reflect"

	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// ProcessProposerSlashings is one of the operations performed
// on each processed beacon block to penalize proposers based on
// slashing conditions if any slashable events occurred.
//
// Official spec definition for proposer slashins:
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
// Verify that len(block.body.casper_slashings) <= MAX_CASPER_SLASHINGS.
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
		// verifyCasperSlashing performs validity checks on the slashing
		// struct and returns a list of validator indices to be slashed from
		// the BeaconState's validator registry.
		validatorIndices, err := verifyCasperSlashing(validatorRegistry, slashing)
		if err != nil {
			return nil, fmt.Errorf("could not verify casper slashing #%d: %v", idx, err)
		}
		for _, validatorIndex := range validatorIndices {
			proposer := validatorRegistry[validatorIndex]
			if proposer.Status != pb.ValidatorRecord_EXITED_WITH_PENALTY {
				// TODO(#781): Replace with
				// update_validator_status(
				//   state,
				//   validatorIndex,
				//   new_status=EXITED_WITH_PENALTY,
				// ) after update_validator_status is implemented.
				validatorRegistry[validatorIndex] = v.ExitValidator(
					proposer,
					currentSlot,
					true, /* penalize */
				)
			}
		}
	}
	return nil, nil
}

func verifyCasperSlashing(
	validatorRegistry []*pb.ValidatorRecord,
	slashing *pb.CasperSlashing,
) ([]uint32, error) {
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

	votes1Attestation := votes1.GetData()
	votes2Attestation := votes2.GetData()

	if err := verifyCasperVotes(validatorRegistry, votes1); err != nil {
		return nil, fmt.Errorf("could not verify casper votes 1: %v", err)
	}
	if err := verifyCasperVotes(validatorRegistry, votes2); err != nil {
		return nil, fmt.Errorf("could not verify casper votes 2: %v", err)
	}

	// Inner attestation data structures for the votes should not be equal.
	if reflect.DeepEqual(votes1Attestation, votes2Attestation) {
		return nil, fmt.Errorf(
			"casper slashing inner vote attestation data should not match: %v, %v",
			votes1Attestation,
			votes2Attestation,
		)
	}

	// Unless vote1.justified_slot < vote2.justified_slot == vote2.slot < vote1.slot or slots
	// for both are strictly strict equal, the slashing is invalid.
	voteJustifiedSlotsLessThan := votes1Attestation.GetJustifiedSlot()+1 <
		votes2Attestation.GetJustifiedSlot()+1
	slotsLessThan := votes2Attestation.GetSlot() < votes1Attestation.GetSlot()
	slotsEqual := votes1Attestation.GetSlot() == votes2Attestation.GetSlot()

	if !(voteJustifiedSlotsLessThan == slotsLessThan) && !slotsEqual {
		return nil, fmt.Errorf(
			`
			expected vote1.JustifiedSlot < vote2.JustifiedSlot == vote2.slot < vote1.slot
			or vote1.slot == vote2.slot, instead received vote1.JustifiedSlot = %d,
			vote2.JustifiedSlot = %d, vote1.slot = %d, and vote2.slot = %d
			`,
			votes1Attestation.GetJustifiedSlot(),
			votes2Attestation.GetJustifiedSlot(),
			votes1Attestation.GetSlot(),
			votes2Attestation.GetSlot(),
		)
	}

	// Obtain the intersection of validator indices between
	// vote1 and vote2's aggregate proof of custody indices.
	indicesIntersection := intersection(votes1Indices, votes2Indices)
	if len(indicesIntersection) < 1 {
		return nil, fmt.Errorf(
			"expected intersection of vote indices to be non-empty: %v",
			indicesIntersection,
		)
	}
	return indicesIntersection, nil
}

func verifyCasperVotes(
	validatorRegistry []*pb.ValidatorRecord,
	votes *pb.SlashableVoteData,
) error {
	totalProofsOfCustody := len(votes.GetAggregateSignaturePoc_0Indices()) +
		len(votes.GetAggregateSignaturePoc_1Indices())
	if uint64(totalProofsOfCustody) > params.BeaconConfig().MaxCasperVotes {
		return fmt.Errorf(
			`
			total proof of custody validator indices (%d) greater than maximum
			allowed number of casper votes (%d)
			`,
			totalProofsOfCustody,
			params.BeaconConfig().MaxCasperVotes,
		)
	}
	_ = validatorRegistry
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

// Computes intersection of two slices with time
// complexity of approximately O(n) leveraging a hash map to
// check for element existence off by a constant factor
// of underlying hash map efficiency.
func intersection(a []uint32, b []uint32) []uint32 {
	set := make([]uint32, 0)
	hash := make(map[uint32]bool)

	for i := 0; i < len(a); i++ {
		hash[a[i]] = true
	}

	for i := 0; i < len(b); i++ {
		if _, found := hash[b[i]]; found {
			set = append(set, b[i])
		}
	}

	return set
}
