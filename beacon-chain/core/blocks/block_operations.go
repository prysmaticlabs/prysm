package blocks

import (
	"bytes"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/types"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
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
			validatorRegistry[slashing.GetProposerIndex()] = v.ExitValidator(proposer, currentSlot, true /* penalize */)
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

// ProcessAttestations applies processing operations to a block's inner attestation
// records. This function returns a list of pending attestations which can then be
// appended to the BeaconState's latest attestations.
//
// Official spec definition for proposer slashings:
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
func ProcessAttestations(
	beaconState *types.BeaconState,
	block *types.Block,
) ([]*pb.PendingAttestationRecord, error) {
	return nil, nil
}
