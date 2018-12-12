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
		validatorIndices, err := verifyCasperSlashing(slashing)
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

func verifyCasperSlashing(slashing *pb.CasperSlashing) ([]uint32, error) {
	vote1 := slashing.GetVotes_1()
	vote2 := slashing.GetVotes_2()
	vote1Indices := append(
		vote1.GetAggregateSignaturePoc_0Indices(),
		vote1.GetAggregateSignaturePoc_1Indices()...,
	)
	vote2Indices := append(
		vote2.GetAggregateSignaturePoc_0Indices(),
		vote2.GetAggregateSignaturePoc_1Indices()...,
	)

	// TODO:
	// Verify that verify_casper_votes(state, casper_slashing.votes_1).
	// Verify that verify_casper_votes(state, casper_slashing.votes_2).

	if !reflect.DeepEqual(vote1.GetData(), vote2.GetData()) {
		return nil, fmt.Errorf(
			"casper slashing inner vote data does not match: %v, %v",
			vote1,
			vote2,
		)
	}

	indicesIntersection := intersection(vote1Indices, vote2Indices)
	if len(indicesIntersection) < 1 {
		return nil, fmt.Errorf(
			"expected intersection of vote indices to be non-empty: %v",
			indicesIntersection,
		)
	}
	return indicesIntersection, nil
}

// Computes intersection of two sets with time
// complexity of approximately O(n) leveraging a hash map to
// check for element existence off by a constant factor
// of hash map efficiency.
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
