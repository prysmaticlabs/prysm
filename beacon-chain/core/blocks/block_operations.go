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
	"sort"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	"github.com/sirupsen/logrus"
)

// VerifyProposerSignature uses BLS signature verification to ensure
// the correct proposer created an incoming beacon block during state
// transition processing.
//
// WIP - this is stubbed out until BLS is integrated into Prysm.
func VerifyProposerSignature(
	_ *pb.BeaconBlock,
) error {

	return nil
}

// ProcessEth1DataInBlock is an operation performed on each
// beacon block to ensure the ETH1 data votes are processed
// into the beacon state.
//
// Official spec definition of ProcessEth1Data
//   If block.eth1_data equals eth1_data_vote.eth1_data for some eth1_data_vote
//   in state.eth1_data_votes, set eth1_data_vote.vote_count += 1.
//   Otherwise, append to state.eth1_data_votes a new Eth1DataVote(eth1_data=block.eth1_data, vote_count=1).
func ProcessEth1DataInBlock(beaconState *pb.BeaconState, block *pb.BeaconBlock) *pb.BeaconState {
	var eth1DataVoteAdded bool

	for idx := range beaconState.Eth1DataVotes {
		if proto.Equal(beaconState.Eth1DataVotes[idx].Eth1Data, block.Eth1Data) {
			beaconState.Eth1DataVotes[idx].VoteCount++
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

// ProcessRandao checks the block proposer's
// randao commitment and generates a new randao mix to update
// in the beacon state's latest randao mixes slice.
//
// Spec pseudocode definition:
//   def process_randao(state: BeaconState, block: BeaconBlock) -> None:
//     proposer = state.validator_registry[get_beacon_proposer_index(state)]
//     # Verify that the provided randao value is valid
//     assert bls_verify(proposer.pubkey, hash_tree_root(get_current_epoch(state)), block.body.randao_reveal, get_domain(state, DOMAIN_RANDAO))
//     # Mix it in
//     state.latest_randao_mixes[get_current_epoch(state) % LATEST_RANDAO_MIXES_LENGTH] = (
//         xor(get_randao_mix(state, get_current_epoch(state)),
//             hash(block.body.randao_reveal))
//     )
func ProcessRandao(
	beaconState *pb.BeaconState,
	block *pb.BeaconBlock,
	verifySignatures bool,
	enableLogging bool,
) (*pb.BeaconState, error) {
	if verifySignatures {
		proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
		if err != nil {
			return nil, fmt.Errorf("could not get beacon proposer index: %v", err)
		}

		if err := verifyBlockRandao(beaconState, block, proposerIdx, enableLogging); err != nil {
			return nil, fmt.Errorf("could not verify block randao: %v", err)
		}
	}
	// If block randao passed verification, we XOR the state's latest randao mix with the block's
	// randao and update the state's corresponding latest randao mix value.
	latestMixesLength := params.BeaconConfig().LatestRandaoMixesLength
	currentEpoch := helpers.CurrentEpoch(beaconState)
	latestMixSlice := beaconState.LatestRandaoMixes[currentEpoch%latestMixesLength]
	blockRandaoReveal := hashutil.Hash(block.Body.RandaoReveal)
	for i, x := range blockRandaoReveal {
		latestMixSlice[i] ^= x
	}
	beaconState.LatestRandaoMixes[currentEpoch%latestMixesLength] = latestMixSlice
	return beaconState, nil
}

// Verify that bls_verify(proposer.pubkey, hash_tree_root(get_current_epoch(state)),
//   block.body.randao_reveal, domain=get_domain(state.fork, get_current_epoch(state), DOMAIN_RANDAO))
func verifyBlockRandao(beaconState *pb.BeaconState, block *pb.BeaconBlock, proposerIdx uint64, enableLogging bool) error {
	proposer := beaconState.ValidatorRegistry[proposerIdx]
	pub, err := bls.PublicKeyFromBytes(proposer.Pubkey)
	if err != nil {
		return fmt.Errorf("could not deserialize proposer public key: %v", err)
	}
	currentEpoch := helpers.CurrentEpoch(beaconState)
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, currentEpoch)
	domain := helpers.DomainVersion(beaconState, currentEpoch, params.BeaconConfig().DomainRandao)
	sig, err := bls.SignatureFromBytes(block.Body.RandaoReveal)
	if err != nil {
		return fmt.Errorf("could not deserialize block randao reveal: %v", err)
	}
	if enableLogging {
		log.WithFields(logrus.Fields{
			"epoch":         helpers.CurrentEpoch(beaconState) - params.BeaconConfig().GenesisEpoch,
			"proposerIndex": proposerIdx,
		}).Info("Verifying randao")
	}
	if !sig.Verify(buf, pub, domain) {
		return fmt.Errorf("block randao reveal signature did not verify")
	}
	return nil
}

// ProcessProposerSlashings is one of the operations performed
// on each processed beacon block to slash proposers based on
// slashing conditions if any slashable events occurred.
//
// Official spec definition for proposer slashings:
//   Verify that len(block.body.proposer_slashings) <= MAX_PROPOSER_SLASHINGS.
//
//   For each proposer_slashing in block.body.proposer_slashings:
//     proposer = state.validator_registry[proposer_slashing.proposer_index]
//     # Verify that the epoch is the same
//     assert slot_to_epoch(proposer_slashing.header_1.slot) == slot_to_epoch(proposer_slashing.header_2.slot)
//     # But the headers are different
//     assert proposer_slashing.header_1 != proposer_slashing.header_2
//     # Check proposer is slashable
//     assert is_slashable_validator(proposer, get_current_epoch(state))
//     # Signatures are valid
//     for header in (proposer_slashing.header_1, proposer_slashing.header_2):
//       domain = get_domain(state, DOMAIN_BEACON_PROPOSER, slot_to_epoch(header.slot))
//       assert bls_verify(proposer.pubkey, signing_root(header), header.signature, domain)
//     slash_validator(state, proposer_slashing.proposer_index)
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
		proposer := registry[slashing.ProposerIndex]
		if err = verifyProposerSlashing(beaconState, proposer, slashing, verifySignatures); err != nil {
			return nil, fmt.Errorf("could not verify proposer slashing %d: %v", idx, err)
		}
		beaconState, err = v.SlashValidator(
			beaconState, slashing.ProposerIndex, 0, /* proposer is whistleblower */
		)
		if err != nil {
			return nil, fmt.Errorf("could not slash proposer index %d: %v",
				slashing.ProposerIndex, err)
		}
	}
	return beaconState, nil
}

func verifyProposerSlashing(
	beaconState *pb.BeaconState,
	proposer *pb.Validator,
	slashing *pb.ProposerSlashing,
	verifySignatures bool,
) error {
	headerEpoch1 := helpers.SlotToEpoch(slashing.Header_1.Slot)
	headerEpoch2 := helpers.SlotToEpoch(slashing.Header_2.Slot)
	if headerEpoch1 != headerEpoch2 {
		return fmt.Errorf("mismatched header epochs, received %d == %d", headerEpoch1, headerEpoch2)
	}
	if proto.Equal(slashing.Header_1, slashing.Header_2) {
		return errors.New("expected slashing headers to differ")
	}
	if !helpers.IsSlashableValidator(proposer, helpers.CurrentEpoch(beaconState)) {
		return fmt.Errorf("validator with key %#x is not slashable", proposer.Pubkey)
	}
	if verifySignatures {
		// TODO(#258): Implement BLS verify of header signatures.
		return nil
	}
	return nil
}

// ProcessAttesterSlashings is one of the operations performed
// on each processed beacon block to slash attesters based on
// Casper FFG slashing conditions if any slashable events occurred.
//
// Official spec definition for attester slashings:
//
//   Verify that len(block.body.attester_slashings) <= MAX_ATTESTER_SLASHINGS.
//
//   For each attester_slashing in block.body.attester_slashings:
//     Let slashable_attestation_1 = attester_slashing.slashable_attestation_1.
//     Let slashable_attestation_2 = attester_slashing.slashable_attestation_2.
//     Verify that slashable_attestation_1.data != slashable_attestation_2.data.
//     Verify that is_double_vote(slashable_attestation_1.data, slashable_attestation_2.data)
//       or is_surround_vote(slashable_attestation_1.data, slashable_attestation_2.data).
//     Verify that verify_slashable_attestation(state, slashable_attestation_1).
//     Verify that verify_slashable_attestation(state, slashable_attestation_2).
//     Let slashable_indices = [index for index in slashable_attestation_1.validator_indices if
//       index in slashable_attestation_2.validator_indices and
//       state.validator_registry[index].slashed_epoch > get_current_epoch(state)].
//     Verify that len(slashable_indices) >= 1.
//     Run slash_validator(state, index) for each index in slashable_indices.
func ProcessAttesterSlashings(
	beaconState *pb.BeaconState,
	block *pb.BeaconBlock,
	verifySignatures bool,
) (*pb.BeaconState, error) {
	body := block.Body
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
		slashableIndices := slashableAttesterIndices(slashing)
		currentEpoch := helpers.CurrentEpoch(beaconState)
		var err error
		var slashedAny bool
		for _, validatorIndex := range slashableIndices {
			if helpers.IsSlashableValidator(beaconState.ValidatorRegistry[validatorIndex], currentEpoch) {
				beaconState, err = v.SlashValidator(beaconState, validatorIndex, 0)
				if err != nil {
					return nil, fmt.Errorf("could not slash validator index %d: %v",
						validatorIndex, err)
				}
				slashedAny = true
			}
		}
		if !slashedAny {
			return nil, errors.New("unable to slash any validator despite confirmed attester slashing")
		}
	}
	return beaconState, nil
}

func verifyAttesterSlashing(slashing *pb.AttesterSlashing, verifySignatures bool) error {
	att1 := slashing.Attestation_1
	att2 := slashing.Attestation_2
	data1 := att1.Data
	data2 := att2.Data
	if !isSlashableAttestationData(data1, data2) {
		return errors.New("attestations are not slashable")
	}
	if err := validateIndexedAttestation(att1, verifySignatures); err != nil {
		return fmt.Errorf("could not validate indexed attestation: %v", err)
	}
	if err := validateIndexedAttestation(att2, verifySignatures); err != nil {
		return fmt.Errorf("could not validate indexed attestation: %v", err)
	}
	return nil
}

// isSlashableAttestationData verifies a slashing against the Casper Proof of Stake FFG rules.
//   return (
//   # Double vote
//   (data_1 != data_2 and data_1.target_epoch == data_2.target_epoch) or
//   # Surround vote
//   (data_1.source_epoch < data_2.source_epoch and data_2.target_epoch < data_1.target_epoch)
//   )
func isSlashableAttestationData(data1 *pb.AttestationData, data2 *pb.AttestationData) bool {
	// Inner attestation data structures for the votes should not be equal,
	// as that would mean both votes are the same and therefore no slashing
	// should occur.
	isDoubleVote := proto.Equal(data1, data2) && data1.TargetEpoch == data2.TargetEpoch
	isSurroundVote := data1.SourceEpoch < data2.SourceEpoch && data2.TargetEpoch < data1.TargetEpoch
	return isDoubleVote || isSurroundVote
}

// validateIndexedAttestation verifies an attestation's custody and bls bit information.
//  """
//    Verify validity of ``indexed_attestation``.
//    """
//    bit_0_indices = indexed_attestation.custody_bit_0_indices
//    bit_1_indices = indexed_attestation.custody_bit_1_indices
//
//    # Verify no index has custody bit equal to 1 [to be removed in phase 1]
//    assert len(bit_1_indices) == 0
//    # Verify max number of indices
//    assert len(bit_0_indices) + len(bit_1_indices) <= MAX_INDICES_PER_ATTESTATION
//    # Verify index sets are disjoint
//    assert len(set(bit_0_indices).intersection(bit_1_indices)) == 0
//    # Verify indices are sorted
//    assert bit_0_indices == sorted(bit_0_indices) and bit_1_indices == sorted(bit_1_indices)
//    # Verify aggregate signature
//    assert bls_verify_multiple(
//        pubkeys=[
//            bls_aggregate_pubkeys([state.validator_registry[i].pubkey for i in bit_0_indices]),
//            bls_aggregate_pubkeys([state.validator_registry[i].pubkey for i in bit_1_indices]),
//        ],
//        message_hashes=[
//            hash_tree_root(AttestationDataAndCustodyBit(data=indexed_attestation.data, custody_bit=0b0)),
//            hash_tree_root(AttestationDataAndCustodyBit(data=indexed_attestation.data, custody_bit=0b1)),
//        ],
//        signature=indexed_attestation.signature,
//        domain=get_domain(state, DOMAIN_ATTESTATION, indexed_attestation.data.target_epoch),
func validateIndexedAttestation(attestation *pb.IndexedAttestation, verifySignatures bool) error {
	bit0Indices := attestation.CustodyBit_0Indices
	bit1Indices := attestation.CustodyBit_1Indices
	if len(bit1Indices) != 0 {
		return fmt.Errorf("expected no bit 1 indices, received %d", len(bit1Indices))
	}
	intersection := sliceutil.IntersectionUint64(bit0Indices, bit1Indices)
	if len(intersection) != 0 {
		return fmt.Errorf("expected disjoint bit indices, received %d bits in common", intersection)
	}
	if uint64(len(bit0Indices)+len(bit1Indices)) > params.BeaconConfig().MaxIndicesPerAttestation {
		return fmt.Errorf("exceeded max number of bit indices: %d", len(bit0Indices)+len(bit1Indices))
	}
	if !sliceutil.IsUint64Sorted(bit0Indices) || !sliceutil.IsUint64Sorted(bit1Indices) {
		return errors.New("bit indices not sorted")
	}
	if verifySignatures {
		// TODO(#258): Implement BLS verify of attestation bit information.
		return nil
	}
	return nil
}

func slashableAttesterIndices(slashing *pb.AttesterSlashing) []uint64 {
	att1 := slashing.Attestation_1
	att2 := slashing.Attestation_1
	indices1 := append(att1.CustodyBit_0Indices, att1.CustodyBit_1Indices...)
	indices2 := append(att2.CustodyBit_0Indices, att2.CustodyBit_1Indices...)
	return sliceutil.IntersectionUint64(indices1, indices2)
}

// ProcessBlockAttestations applies processing operations to a block's inner attestation
// records. This function returns a list of pending attestations which can then be
// appended to the BeaconState's latest attestations.
//
// Official spec definition for block attestation processing:
//   Verify that len(block.body.attestations) <= MAX_ATTESTATIONS.
//
//   For each attestation in block.body.attestations:
//     Verify that `attestation.data.slot >= GENESIS_SLOT`.
//     Verify that `attestation.data.slot + MIN_ATTESTATION_INCLUSION_DELAY <= state.slot`.
//     Verify that `state.slot < attestation.data.slot + SLOTS_PER_EPOCH.
//     Verify that attestation.data.justified_epoch is equal to state.justified_epoch
//       if attestation.data.slot >= get_epoch_start_slot(get_current_epoch(state)) else state.previous_justified_epoch.
//     Verify that attestation.data.justified_block_root is equal to
//       get_block_root(state, get_epoch_start_slot(attestation.data.justified_epoch)).
//     Verify that either attestation.data.latest_crosslink_root or
//       attestation.data.shard_block_root equals state.latest_crosslinks[shard].shard_block_root.
//     Verify bitfields and aggregate signature using BLS.
//     [TO BE REMOVED IN PHASE 1] Verify that attestation.data.shard_block_root == ZERO_HASH.
//     Append PendingAttestation(data=attestation.data, aggregation_bitfield=attestation.aggregation_bitfield,
//       custody_bitfield=attestation.custody_bitfield, inclusion_slot=state.slot) to state.latest_attestations
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

	for idx, attestation := range atts {
		if err := VerifyAttestation(beaconState, attestation, verifySignatures); err != nil {
			return nil, fmt.Errorf("could not verify attestation at index %d in block: %v", idx, err)
		}

		beaconState.LatestAttestations = append(beaconState.LatestAttestations, &pb.PendingAttestation{
			Data:                attestation.Data,
			AggregationBitfield: attestation.AggregationBitfield,
			CustodyBitfield:     attestation.CustodyBitfield,
			InclusionSlot:       beaconState.Slot,
		})
	}

	return beaconState, nil
}

// VerifyAttestation verifies an input attestation can pass through processing using the given beacon state.
func VerifyAttestation(beaconState *pb.BeaconState, att *pb.Attestation, verifySignatures bool) error {
	if att.Data.Slot < params.BeaconConfig().GenesisSlot {
		return fmt.Errorf(
			"attestation slot (slot %d) less than genesis slot (%d)",
			att.Data.Slot,
			params.BeaconConfig().GenesisSlot,
		)
	}
	inclusionDelay := params.BeaconConfig().MinAttestationInclusionDelay
	if att.Data.Slot+inclusionDelay > beaconState.Slot {
		return fmt.Errorf(
			"attestation slot (slot %d) + inclusion delay (%d) beyond current beacon state slot (%d)",
			att.Data.Slot-params.BeaconConfig().GenesisSlot,
			inclusionDelay,
			beaconState.Slot-params.BeaconConfig().GenesisSlot,
		)
	}
	if att.Data.Slot+params.BeaconConfig().SlotsPerEpoch < beaconState.Slot {
		return fmt.Errorf(
			"attestation slot (slot %d) + epoch length (%d) less than current beacon state slot (%d)",
			att.Data.Slot-params.BeaconConfig().GenesisSlot,
			params.BeaconConfig().SlotsPerEpoch,
			beaconState.Slot-params.BeaconConfig().GenesisSlot,
		)
	}
	// Verify that `attestation.data.justified_epoch` is equal to `state.justified_epoch
	// and verify that `attestation.data.justified_root` is equal to `state.justified_root
	// 	if slot_to_epoch(attestation.data.slot + 1) >= get_current_epoch(state)
	// 	else state.previous_justified_epoch`.
	if helpers.SlotToEpoch(att.Data.Slot+1) >= helpers.CurrentEpoch(beaconState) {
		if att.Data.JustifiedEpoch != beaconState.CurrentJustifiedEpoch {
			return fmt.Errorf(
				"expected attestation.JustifiedEpoch == state.CurrentJustifiedEpoch, received %d == %d",
				att.Data.JustifiedEpoch-params.BeaconConfig().GenesisEpoch,
				beaconState.CurrentJustifiedEpoch-params.BeaconConfig().GenesisEpoch,
			)
		}

		if !bytes.Equal(att.Data.JustifiedBlockRootHash32, beaconState.CurrentJustifiedRoot) {
			return fmt.Errorf(
				"expected attestation.JustifiedRoot == state.CurrentJustifiedRoot, received %#x == %#x",
				att.Data.JustifiedBlockRootHash32,
				beaconState.CurrentJustifiedRoot,
			)
		}
	} else {
		if att.Data.JustifiedEpoch != beaconState.PreviousJustifiedEpoch {
			return fmt.Errorf(
				"expected attestation.JustifiedEpoch == state.PreviousJustifiedEpoch, received %d == %d",
				att.Data.JustifiedEpoch-params.BeaconConfig().GenesisEpoch,
				beaconState.PreviousJustifiedEpoch-params.BeaconConfig().GenesisEpoch,
			)
		}
		if !bytes.Equal(att.Data.JustifiedBlockRootHash32, beaconState.PreviousJustifiedRoot) {
			return fmt.Errorf(
				"expected attestation.JustifiedRoot == state.PreviousJustifiedRoot, received %#x == %#x",
				att.Data.JustifiedBlockRootHash32,
				beaconState.CurrentJustifiedRoot,
			)
		}
	}
	// Verify that either:
	// 1.) Crosslink(shard_block_root=attestation.data.shard_block_root,
	// 	epoch=slot_to_epoch(attestation.data.slot)) equals
	// 	state.latest_crosslinks[attestation.data.shard]
	// 2.) attestation.data.latest_crosslink
	// 	equals state.latest_crosslinks[attestation.data.shard]
	shard := att.Data.Shard
	crosslink := &pb.Crosslink{
		CrosslinkDataRootHash32: att.Data.CrosslinkDataRoot,
		Epoch:                   helpers.SlotToEpoch(att.Data.Slot),
	}
	crosslinkFromAttestation := att.Data.LatestCrosslink
	crosslinkFromState := beaconState.LatestCrosslinks[shard]

	if !(reflect.DeepEqual(crosslinkFromState, crosslink) ||
		reflect.DeepEqual(crosslinkFromState, crosslinkFromAttestation)) {
		return fmt.Errorf(
			"incoming attestation does not match crosslink in state for shard %d",
			shard,
		)
	}

	// Verify attestation.shard_block_root == ZERO_HASH [TO BE REMOVED IN PHASE 1].
	if !bytes.Equal(att.Data.CrosslinkDataRoot, params.BeaconConfig().ZeroHash[:]) {
		return fmt.Errorf(
			"expected attestation.data.CrosslinkDataRootHash == %#x, received %#x instead",
			params.BeaconConfig().ZeroHash[:],
			att.Data.CrosslinkDataRoot,
		)
	}
	if verifySignatures {
		// TODO(#258): Integrate BLS signature verification for attestation.
		// assert bls_verify_multiple(
		//   pubkeys=[
		//	 bls_aggregate_pubkeys([state.validator_registry[i].pubkey for i in custody_bit_0_participants]),
		//   bls_aggregate_pubkeys([state.validator_registry[i].pubkey for i in custody_bit_1_participants]),
		//   ],
		//   message_hash=[
		//   hash_tree_root(AttestationDataAndCustodyBit(data=attestation.data, custody_bit=0b0)),
		//   hash_tree_root(AttestationDataAndCustodyBit(data=attestation.data, custody_bit=0b1)),
		//   ],
		//   signature=attestation.aggregate_signature,
		//   domain=get_domain(state.fork, slot_to_epoch(attestation.data.slot), DOMAIN_ATTESTATION),
		// )
		return nil
	}
	return nil
}

// ConvertToIndexed converts attestation to (almost) indexed-verifiable form.
//
// Spec pseudocode definition:
//   def convert_to_indexed(state: BeaconState, attestation: Attestation) -> IndexedAttestation:
//     """
//     Convert ``attestation`` to (almost) indexed-verifiable form.
//     """
//     attesting_indices = get_attesting_indices(state, attestation.data, attestation.aggregation_bitfield)
//     custody_bit_1_indices = get_attesting_indices(state, attestation.data, attestation.custody_bitfield)
//     custody_bit_0_indices = [index for index in attesting_indices if index not in custody_bit_1_indices]
//     return IndexedAttestation(
//         custody_bit_0_indices=custody_bit_0_indices,
//         custody_bit_1_indices=custody_bit_1_indices,
//         data=attestation.data,
//         signature=attestation.signature,
//     )
func ConvertToIndexed(state *pb.BeaconState, attestation *pb.Attestation) (*pb.IndexedAttestation, error) {
	attI, err := helpers.AttestingIndices(state, attestation.Data, attestation.AggregationBitfield)
	if err != nil {
		return nil, err
	}
	cb1i, err := helpers.AttestingIndices(state, attestation.Data, attestation.CustodyBitfield)
	if err != nil {
		return nil, err
	}
	cb1iMap := make(map[uint64]bool)
	for _, in := range cb1i {
		cb1iMap[in] = true
	}
	cb0i := []uint64{}
	for _, index := range attI {
		_, ok := cb1iMap[index]
		if !ok {
			cb0i = append(cb0i, index)
		}
	}
	inAtt := &pb.IndexedAttestation{
		Data:                attestation.Data,
		Signature:           attestation.Signature,
		CustodyBit_0Indices: cb0i,
		CustodyBit_1Indices: cb1i,
	}
	return inAtt, nil
}

// VerifyIndexedAttestation determines the validity of an indexed attestation.
// WIP - signing is not implemented until BLS is integrated into Prysm.
//
// Spec pseudocode definition:
//  def verify_indexed_attestation(state: BeaconState, indexed_attestation: IndexedAttestation) -> bool:
//    """
//    Verify validity of ``indexed_attestation`` fields.
//    """
//    custody_bit_0_indices = indexed_attestation.custody_bit_0_indices
//    custody_bit_1_indices = indexed_attestation.custody_bit_1_indices
//
//    # Ensure no duplicate indices across custody bits
//    assert len(set(custody_bit_0_indices).intersection(set(custody_bit_1_indices))) == 0
//
//    if len(custody_bit_1_indices) > 0:  # [TO BE REMOVED IN PHASE 1]
//        return False
//
//    if not (1 <= len(custody_bit_0_indices) + len(custody_bit_1_indices) <= MAX_INDICES_PER_ATTESTATION):
//        return False
//
//    if custody_bit_0_indices != sorted(custody_bit_0_indices):
//        return False
//
//    if custody_bit_1_indices != sorted(custody_bit_1_indices):
//        return False
//
//    return bls_verify_multiple(
//        pubkeys=[
//            bls_aggregate_pubkeys([state.validator_registry[i].pubkey for i in custody_bit_0_indices]),
//            bls_aggregate_pubkeys([state.validator_registry[i].pubkey for i in custody_bit_1_indices]),
//        ],
//        message_hashes=[
//            hash_tree_root(AttestationDataAndCustodyBit(data=indexed_attestation.data, custody_bit=0b0)),
//            hash_tree_root(AttestationDataAndCustodyBit(data=indexed_attestation.data, custody_bit=0b1)),
//        ],
//        signature=indexed_attestation.signature,
//        domain=get_domain(state, DOMAIN_ATTESTATION, slot_to_epoch(indexed_attestation.data.slot)),
//    )
func VerifyIndexedAttestation(state *pb.BeaconState, indexedAtt *pb.IndexedAttestation) (bool, error) {
	custodyBit0Indices := indexedAtt.CustodyBit_0Indices
	custodyBit1Indices := indexedAtt.CustodyBit_1Indices

	custodyBitIntersection := sliceutil.IntersectionUint64(custodyBit0Indices, custodyBit1Indices)
	if len(custodyBitIntersection) != 0 {
		return false, fmt.Errorf("custody bit indice should not contain duplicates, received: %v", custodyBitIntersection)
	}

	// To be removed in phase 1
	if len(custodyBit1Indices) > 0 {
		return false, nil
	}

	maxIndices := params.BeaconConfig().MaxIndicesPerAttestation
	totalIndicesLength := uint64(len(custodyBit0Indices) + len(custodyBit1Indices))
	if maxIndices < totalIndicesLength || totalIndicesLength < 1 {
		return false, nil
	}

	if !sort.SliceIsSorted(custodyBit0Indices, func(i, j int) bool {
		return custodyBit0Indices[i] < custodyBit0Indices[j]
	}) {
		return false, nil
	}

	return true, nil
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
//     Verify deposit merkle_branch, setting leaf=hash(serialized_deposit_data), branch=deposit.branch,
//     depth=DEPOSIT_CONTRACT_TREE_DEPTH and root=state.latest_eth1_data.deposit_root, index = deposit.index:
//
//     Run the following:
//     process_deposit(
//       state=state,
//       pubkey=deposit.deposit_data.deposit_input.pubkey,
//       deposit=deposit.deposit_data.value,
//       proof_of_possession=deposit.deposit_data.deposit_input.proof_of_possession,
//       withdrawal_credentials=deposit.deposit_data.deposit_input.withdrawal_credentials,
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
		depositInput, err = helpers.DecodeDepositInput(depositData)
		if err != nil {
			beaconState = processInvalidDeposit(beaconState)
			log.Errorf("could not decode deposit input: %v", err)
			continue
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
			binary.LittleEndian.Uint64(depositValue),
			depositInput.ProofOfPossession,
			depositInput.WithdrawalCredentialsHash32,
		)
		if err != nil {
			beaconState = processInvalidDeposit(beaconState)
			log.Errorf("could not process deposit into beacon state: %v", err)
			continue
		}
	}
	return beaconState, nil
}

func verifyDeposit(beaconState *pb.BeaconState, deposit *pb.Deposit) error {
	// Deposits must be processed in order
	if deposit.Index != beaconState.DepositIndex {
		return fmt.Errorf(
			"expected deposit merkle tree index to match beacon state deposit index, wanted: %d, received: %d",
			beaconState.DepositIndex,
			deposit.Index,
		)
	}

	// Verify Merkle proof of deposit and deposit trie root.
	receiptRoot := beaconState.LatestEth1Data.DepositRoot
	if ok := trieutil.VerifyMerkleProof(
		receiptRoot,
		deposit.DepositData,
		int(deposit.Index),
		deposit.Proof,
	); !ok {
		return fmt.Errorf(
			"deposit merkle branch of deposit root did not verify for root: %#x",
			receiptRoot,
		)
	}

	return nil
}

// we increase the state deposit index, since deposits have to be processed
// in order even if they are invalid
func processInvalidDeposit(bState *pb.BeaconState) *pb.BeaconState {
	bState.DepositIndex++
	return bState
}

// ProcessValidatorExits is one of the operations performed
// on each processed beacon block to determine which validators
// should exit the state's validator registry.
//
// Official spec definition for processing exits:
//
//   Verify that len(block.body.voluntary_exits) <= MAX_VOLUNTARY_EXITS.
//
//   For each exit in block.body.voluntary_exits:
//     validator = state.validator_registry[exit.validator_index]
//     # Verify the validator is active
//     assert is_active_validator(validator, get_current_epoch(state))
//     # Verify the validator has not yet exited
//     assert validator.exit_epoch == FAR_FUTURE_EPOCH
//     # Exits must specify an epoch when they become valid; they are not valid before then
//     assert get_current_epoch(state) >= exit.epoch
//     # Verify the validator has been active long enough
//     assert get_current_epoch(state) >= validator.activation_epoch + PERSISTENT_COMMITTEE_PERIOD
//     # Verify signature
//     domain = get_domain(state, DOMAIN_VOLUNTARY_EXIT, exit.epoch)
//     assert bls_verify(validator.pubkey, signing_root(exit), exit.signature, domain)
//     # Initiate exit
//     initiate_validator_exit(state, exit.validator_index)
func ProcessValidatorExits(
	beaconState *pb.BeaconState,
	block *pb.BeaconBlock,
	verifySignatures bool,
) (*pb.BeaconState, error) {

	exits := block.Body.VoluntaryExits
	if uint64(len(exits)) > params.BeaconConfig().MaxVoluntaryExits {
		return nil, fmt.Errorf(
			"number of exits (%d) exceeds allowed threshold of %d",
			len(exits),
			params.BeaconConfig().MaxVoluntaryExits,
		)
	}

	for idx, exit := range exits {
		if err := verifyExit(beaconState, exit, verifySignatures); err != nil {
			return nil, fmt.Errorf("could not verify exit #%d: %v", idx, err)
		}
		beaconState = v.InitiateValidatorExit(beaconState, exit.ValidatorIndex)
	}
	return beaconState, nil
}

func verifyExit(beaconState *pb.BeaconState, exit *pb.VoluntaryExit, verifySignatures bool) error {
	validator := beaconState.ValidatorRegistry[exit.ValidatorIndex]
	currentEpoch := helpers.CurrentEpoch(beaconState)
	// Verify the validator is active.
	if !helpers.IsActiveValidator(validator, currentEpoch) {
		return errors.New("non-active validator cannot exit")
	}
	// Verify the validator has not yet exited.
	if validator.ExitEpoch != params.BeaconConfig().FarFutureEpoch {
		return fmt.Errorf("validator has already exited at epoch: %v", validator.ExitEpoch)
	}
	// Exits must specify an epoch when they become valid; they are not valid before then.
	if currentEpoch < exit.Epoch {
		return fmt.Errorf("expected current epoch >= exit epoch, received %d < %d", currentEpoch, exit.Epoch)
	}
	// Verify the validator has been active long enough.
	if currentEpoch < validator.ActivationEpoch+params.BeaconConfig().PersistentCommitteePeriod {
		return fmt.Errorf(
			"validator has not been active long enough to exit, wanted epoch %d >= %d",
			currentEpoch,
			validator.ActivationEpoch+params.BeaconConfig().PersistentCommitteePeriod,
		)
	}
	if verifySignatures {
		// TODO(#258): Integrate BLS signature verification for exits.
		// domain = get_domain(state, DOMAIN_VOLUNTARY_EXIT, exit.epoch)
		// assert bls_verify(validator.pubkey, signing_root(exit), exit.signature, domain)
		return nil
	}
	return nil
}
