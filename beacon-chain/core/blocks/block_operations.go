package blocks

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	"github.com/sirupsen/logrus"
)

var eth1DataCache = cache.NewEth1DataVoteCache()

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
// Official spec definition:
//   def process_eth1_data(state: BeaconState, body: BeaconBlockBody) -> None:
//    state.eth1_data_votes.append(body.eth1_data)
//    if state.eth1_data_votes.count(body.eth1_data) * 2 > SLOTS_PER_ETH1_VOTING_PERIOD:
//        state.latest_eth1_data = body.eth1_data
func ProcessEth1DataInBlock(beaconState *pb.BeaconState, block *pb.BeaconBlock) (*pb.BeaconState, error) {
	beaconState.Eth1DataVotes = append(beaconState.Eth1DataVotes, block.Body.Eth1Data)

	voteCount, err := eth1DataCache.Eth1DataVote(block.Body.Eth1Data.DepositRoot)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve eth1 data vote cache: %v", err)
	}

	if voteCount == 0 {
		for _, vote := range beaconState.Eth1DataVotes {
			if proto.Equal(vote, block.Body.Eth1Data) {
				voteCount++
			}
		}
	} else {
		voteCount++
	}

	if err := eth1DataCache.AddEth1DataVote(&cache.Eth1DataVote{
		DepositRoot: block.Body.Eth1Data.DepositRoot,
		VoteCount:   voteCount,
	}); err != nil {
		return nil, fmt.Errorf("could not save eth1 data vote cache: %v", err)
	}

	if voteCount*2 > params.BeaconConfig().SlotsPerEth1VotingPeriod {
		beaconState.Eth1Data = block.Body.Eth1Data
	}

	return beaconState, nil
}

// ProcessBlockHeader validates a block by its header.
//
// Spec pseudocode definition:
//
//  def process_block_header(state: BeaconState, block: BeaconBlock) -> None:
//     # Verify that the slots match
//     assert block.slot == state.slot
//     # Verify that the parent matches
//     assert block.parent_root == signing_root(state.latest_block_header)
//     # Save current block as the new latest block
//     state.latest_block_header = BeaconBlockHeader(
//         slot=block.slot,
//         parent_root=block.parent_root,
//         body_root=hash_tree_root(block.body),
//     )
//     # Verify proposer is not slashed
//     proposer = state.validator_registry[get_beacon_proposer_index(state)]
//     assert not proposer.slashed
//     # Verify proposer signature
//     assert bls_verify(proposer.pubkey, signing_root(block), block.signature, get_domain(state, DOMAIN_BEACON_PROPOSER))
func ProcessBlockHeader(
	beaconState *pb.BeaconState,
	block *pb.BeaconBlock,
) (*pb.BeaconState, error) {
	if beaconState.Slot != block.Slot {
		return nil, fmt.Errorf("state slot: %d is different then block slot: %d", beaconState.Slot, block.Slot)
	}

	parentRoot, err := ssz.SigningRoot(beaconState.LatestBlockHeader)
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(block.ParentRoot, parentRoot[:]) {
		return nil, fmt.Errorf(
			"parent root %#x does not match the latest block header signing root in state %#x",
			block.ParentRoot, parentRoot)
	}

	bodyRoot, err := ssz.HashTreeRoot(block.Body)
	if err != nil {
		return nil, err
	}
	emptySig := make([]byte, 96)
	beaconState.LatestBlockHeader = &pb.BeaconBlockHeader{
		Slot:       block.Slot,
		ParentRoot: block.ParentRoot,
		StateRoot:  params.BeaconConfig().ZeroHash[:],
		BodyRoot:   bodyRoot[:],
		Signature:  emptySig,
	}
	// Verify proposer is not slashed.
	idx, err := helpers.BeaconProposerIndex(beaconState)
	if err != nil {
		return nil, err
	}
	proposer := beaconState.Validators[idx]
	if proposer.Slashed {
		return nil, fmt.Errorf("proposer at index %d was previously slashed", idx)
	}
	// TODO(#2307) Verify proposer signature.
	return beaconState, nil
}

// ProcessRandao checks the block proposer's
// randao commitment and generates a new randao mix to update
// in the beacon state's latest randao mixes slice.
//
// Spec pseudocode definition:
//   def process_randao(state: BeaconState, body: BeaconBlockBody) -> None:
//     proposer = state.validator_registry[get_beacon_proposer_index(state)]
//     # Verify that the provided randao value is valid
//     assert bls_verify(
//         proposer.pubkey,
//         hash_tree_root(get_current_epoch(state)),
//         body.randao_reveal,
//         get_domain(state, DOMAIN_RANDAO),
//     )
//     # Mix it in
//     state.latest_randao_mixes[get_current_epoch(state) % LATEST_RANDAO_MIXES_LENGTH] = (
//         xor(get_randao_mix(state, get_current_epoch(state)),
//             hash(body.randao_reveal))
//     )
func ProcessRandao(
	beaconState *pb.BeaconState,
	body *pb.BeaconBlockBody,
	verifySignatures bool,
	enableLogging bool,
) (*pb.BeaconState, error) {
	if verifySignatures {
		proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
		if err != nil {
			return nil, fmt.Errorf("could not get beacon proposer index: %v", err)
		}

		if err := verifyBlockRandao(beaconState, body, proposerIdx, enableLogging); err != nil {
			return nil, fmt.Errorf("could not verify block randao: %v", err)
		}
	}
	// If block randao passed verification, we XOR the state's latest randao mix with the block's
	// randao and update the state's corresponding latest randao mix value.
	latestMixesLength := params.BeaconConfig().EpochsPerHistoricalVector
	currentEpoch := helpers.CurrentEpoch(beaconState)
	latestMixSlice := beaconState.RandaoMixes[currentEpoch%latestMixesLength]
	blockRandaoReveal := hashutil.Hash(body.RandaoReveal)
	for i, x := range blockRandaoReveal {
		latestMixSlice[i] ^= x
	}
	beaconState.RandaoMixes[currentEpoch%latestMixesLength] = latestMixSlice
	return beaconState, nil
}

// Verify that bls_verify(proposer.pubkey, hash_tree_root(get_current_epoch(state)),
//   block.body.randao_reveal, domain=get_domain(state.fork, get_current_epoch(state), DOMAIN_RANDAO))
func verifyBlockRandao(beaconState *pb.BeaconState, body *pb.BeaconBlockBody, proposerIdx uint64, enableLogging bool) error {
	proposer := beaconState.Validators[proposerIdx]
	pub, err := bls.PublicKeyFromBytes(proposer.Pubkey)
	if err != nil {
		return fmt.Errorf("could not deserialize proposer public key: %v", err)
	}
	currentEpoch := helpers.CurrentEpoch(beaconState)
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, currentEpoch)
	domain := helpers.Domain(beaconState, currentEpoch, params.BeaconConfig().DomainRandao)
	sig, err := bls.SignatureFromBytes(body.RandaoReveal)
	if err != nil {
		return fmt.Errorf("could not deserialize block randao reveal: %v", err)
	}
	if enableLogging {
		log.WithFields(logrus.Fields{
			"epoch":         helpers.CurrentEpoch(beaconState),
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
// Spec pseudocode definition:
//   def process_proposer_slashing(state: BeaconState, proposer_slashing: ProposerSlashing) -> None:
//    """
//    Process ``ProposerSlashing`` operation.
//    """
//    proposer = state.validator_registry[proposer_slashing.proposer_index]
//    # Verify that the epoch is the same
//    assert slot_to_epoch(proposer_slashing.header_1.slot) == slot_to_epoch(proposer_slashing.header_2.slot)
//    # But the headers are different
//    assert proposer_slashing.header_1 != proposer_slashing.header_2
//    # Check proposer is slashable
//    assert is_slashable_validator(proposer, get_current_epoch(state))
//    # Signatures are valid
//    for header in (proposer_slashing.header_1, proposer_slashing.header_2):
//        domain = get_domain(state, DOMAIN_BEACON_PROPOSER, slot_to_epoch(header.slot))
//        assert bls_verify(proposer.pubkey, signing_root(header), header.signature, domain)
//
//    slash_validator(state, proposer_slashing.proposer_index)
func ProcessProposerSlashings(
	beaconState *pb.BeaconState,
	body *pb.BeaconBlockBody,
	verifySignatures bool,
) (*pb.BeaconState, error) {
	var err error
	for idx, slashing := range body.ProposerSlashings {
		proposer := beaconState.Validators[slashing.ProposerIndex]
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
// Spec pseudocode definition:
//   def process_attester_slashing(state: BeaconState, attester_slashing: AttesterSlashing) -> None:
//    """
//    Process ``AttesterSlashing`` operation.
//    """
//    attestation_1 = attester_slashing.attestation_1
//    attestation_2 = attester_slashing.attestation_2
//    assert is_slashable_attestation_data(attestation_1.data, attestation_2.data)
//    validate_indexed_attestation(state, attestation_1)
//    validate_indexed_attestation(state, attestation_2)
//
//    slashed_any = False
//    attesting_indices_1 = attestation_1.custody_bit_0_indices + attestation_1.custody_bit_1_indices
//    attesting_indices_2 = attestation_2.custody_bit_0_indices + attestation_2.custody_bit_1_indices
//    for index in sorted(set(attesting_indices_1).intersection(attesting_indices_2)):
//        if is_slashable_validator(state.validator_registry[index], get_current_epoch(state)):
//            slash_validator(state, index)
//            slashed_any = True
//    assert slashed_any
func ProcessAttesterSlashings(
	beaconState *pb.BeaconState,
	body *pb.BeaconBlockBody,
	verifySignatures bool,
) (*pb.BeaconState, error) {
	for idx, slashing := range body.AttesterSlashings {
		if err := verifyAttesterSlashing(slashing, verifySignatures); err != nil {
			return nil, fmt.Errorf("could not verify attester slashing #%d: %v", idx, err)
		}
		slashableIndices := slashableAttesterIndices(slashing)
		currentEpoch := helpers.CurrentEpoch(beaconState)
		var err error
		var slashedAny bool
		for _, validatorIndex := range slashableIndices {
			if helpers.IsSlashableValidator(beaconState.Validators[validatorIndex], currentEpoch) {
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
	if !IsSlashableAttestationData(data1, data2) {
		return errors.New("attestations are not slashable")
	}
	if err := VerifyIndexedAttestation(att1, verifySignatures); err != nil {
		return fmt.Errorf("could not validate indexed attestation: %v", err)
	}
	if err := VerifyIndexedAttestation(att2, verifySignatures); err != nil {
		return fmt.Errorf("could not validate indexed attestation: %v", err)
	}
	return nil
}

// IsSlashableAttestationData verifies a slashing against the Casper Proof of Stake FFG rules.
//
// Spec pseudocode definition:
//   return (
//   # Double vote
//   (data_1 != data_2 and data_1.target_epoch == data_2.target_epoch) or
//   # Surround vote
//   (data_1.source_epoch < data_2.source_epoch and data_2.target_epoch < data_1.target_epoch)
//   )
func IsSlashableAttestationData(data1 *pb.AttestationData, data2 *pb.AttestationData) bool {
	// Inner attestation data structures for the votes should not be equal,
	// as that would mean both votes are the same and therefore no slashing
	// should occur.
	isDoubleVote := !proto.Equal(data1, data2) && data1.TargetEpoch == data2.TargetEpoch
	isSurroundVote := data1.SourceEpoch < data2.SourceEpoch && data2.TargetEpoch < data1.TargetEpoch
	return isDoubleVote || isSurroundVote
}

func slashableAttesterIndices(slashing *pb.AttesterSlashing) []uint64 {
	att1 := slashing.Attestation_1
	att2 := slashing.Attestation_1
	indices1 := append(att1.CustodyBit_0Indices, att1.CustodyBit_1Indices...)
	indices2 := append(att2.CustodyBit_0Indices, att2.CustodyBit_1Indices...)
	return sliceutil.IntersectionUint64(indices1, indices2)
}

// ProcessAttestations applies processing operations to a block's inner attestation
// records. This function returns a list of pending attestations which can then be
// appended to the BeaconState's latest attestations.
func ProcessAttestations(
	beaconState *pb.BeaconState,
	body *pb.BeaconBlockBody,
	verifySignatures bool,
) (*pb.BeaconState, error) {
	var err error
	for idx, attestation := range body.Attestations {
		beaconState, err = ProcessAttestation(beaconState, attestation, verifySignatures)
		if err != nil {
			return nil, fmt.Errorf("could not verify attestation at index %d in block: %v", idx, err)
		}
	}

	return beaconState, nil
}

// ProcessAttestation verifies an input attestation can pass through processing using the given beacon state.
//
// Spec pseudocode definition:
//  def process_attestation(state: BeaconState, attestation: Attestation) -> None:
//    """
//    Process ``Attestation`` operation.
//    """
//    data = attestation.data
//    attestation_slot = get_attestation_data_slot(state, data)
//    assert attestation_slot + MIN_ATTESTATION_INCLUSION_DELAY <= state.slot <= attestation_slot + SLOTS_PER_EPOCH
//
//    pending_attestation = PendingAttestation(
//        data=data,
//        aggregation_bitfield=attestation.aggregation_bitfield,
//        inclusion_delay=state.slot - attestation_slot,
//        proposer_index=get_beacon_proposer_index(state),
//    )
//
//    assert data.target_epoch in (get_previous_epoch(state), get_current_epoch(state))
//    if data.target_epoch == get_current_epoch(state):
//        ffg_data = (state.current_justified_epoch, state.current_justified_root, get_current_epoch(state))
//        parent_crosslink = state.current_crosslinks[data.crosslink.shard]
//        state.current_epoch_attestations.append(pending_attestation)
//    else:
//        ffg_data = (state.previous_justified_epoch, state.previous_justified_root, get_previous_epoch(state))
//        parent_crosslink = state.previous_crosslinks[data.crosslink.shard]
//        state.previous_epoch_attestations.append(pending_attestation)
//
//    # Check FFG data, crosslink data, and signature
//    assert ffg_data == (data.source_epoch, data.source_root, data.target_epoch)
//    assert data.crosslink.start_epoch == parent_crosslink.end_epoch
//    assert data.crosslink.end_epoch == min(data.target_epoch, parent_crosslink.end_epoch + MAX_EPOCHS_PER_CROSSLINK)
//    assert data.crosslink.parent_root == hash_tree_root(parent_crosslink)
//    assert data.crosslink.data_root == ZERO_HASH  # [to be removed in phase 1]
//    validate_indexed_attestation(state, convert_to_indexed(state, attestation))
func ProcessAttestation(beaconState *pb.BeaconState, att *pb.Attestation, verifySignatures bool) (*pb.BeaconState, error) {
	data := att.Data
	attestationSlot, err := helpers.AttestationDataSlot(beaconState, data)
	if err != nil {
		return nil, fmt.Errorf("could not get attestation slot: %v", err)
	}
	minInclusionCheck := attestationSlot+params.BeaconConfig().MinAttestationInclusionDelay <= beaconState.Slot
	epochInclusionCheck := beaconState.Slot <= attestationSlot+params.BeaconConfig().SlotsPerEpoch

	if !minInclusionCheck {
		return nil, fmt.Errorf(
			"attestation slot %d + inclusion delay %d > state slot %d",
			attestationSlot,
			params.BeaconConfig().MinAttestationInclusionDelay,
			beaconState.Slot,
		)
	}
	if !epochInclusionCheck {
		return nil, fmt.Errorf(
			"state slot %d > attestation slot %d + SLOTS_PER_EPOCH %d",
			beaconState.Slot,
			attestationSlot,
			params.BeaconConfig().SlotsPerEpoch,
		)
	}
	proposerIndex, err := helpers.BeaconProposerIndex(beaconState)
	if err != nil {
		return nil, err
	}
	pendingAtt := &pb.PendingAttestation{
		Data:            data,
		AggregationBits: att.AggregationBits,
		InclusionDelay:  beaconState.Slot - attestationSlot,
		ProposerIndex:   proposerIndex,
	}

	if !(data.TargetEpoch == helpers.PrevEpoch(beaconState) || data.TargetEpoch == helpers.CurrentEpoch(beaconState)) {
		return nil, fmt.Errorf(
			"expected target epoch %d == %d or %d",
			data.TargetEpoch,
			helpers.PrevEpoch(beaconState),
			helpers.CurrentEpoch(beaconState),
		)
	}

	var ffgSourceEpoch uint64
	var ffgSourceRoot []byte
	var ffgTargetEpoch uint64
	var parentCrosslink *pb.Crosslink
	if data.TargetEpoch == helpers.CurrentEpoch(beaconState) {
		ffgSourceEpoch = beaconState.CurrentJustifiedCheckpoint.Epoch
		ffgSourceRoot = beaconState.CurrentJustifiedCheckpoint.Root
		ffgTargetEpoch = helpers.CurrentEpoch(beaconState)
		parentCrosslink = beaconState.CurrentCrosslinks[data.Crosslink.Shard]
		beaconState.CurrentEpochAttestations = append(beaconState.CurrentEpochAttestations, pendingAtt)
	} else {
		ffgSourceEpoch = beaconState.PreviousJustifiedCheckpoint.Epoch
		ffgSourceRoot = beaconState.PreviousJustifiedCheckpoint.Root
		ffgTargetEpoch = helpers.PrevEpoch(beaconState)
		parentCrosslink = beaconState.PreviousCrosslinks[data.Crosslink.Shard]
		beaconState.PreviousEpochAttestations = append(beaconState.PreviousEpochAttestations, pendingAtt)
	}
	if data.SourceEpoch != ffgSourceEpoch {
		return nil, fmt.Errorf("expected source epoch %d, received %d", ffgSourceEpoch, data.SourceEpoch)
	}
	if !bytes.Equal(data.SourceRoot, ffgSourceRoot) {
		return nil, fmt.Errorf("expected source root %#x, received %#x", ffgSourceRoot, data.SourceRoot)
	}
	if data.TargetEpoch != ffgTargetEpoch {
		return nil, fmt.Errorf("expected target epoch %d, received %d", ffgTargetEpoch, data.TargetEpoch)
	}
	endEpoch := parentCrosslink.EndEpoch + params.BeaconConfig().MaxEpochsPerCrosslink
	if data.TargetEpoch < endEpoch {
		endEpoch = data.TargetEpoch
	}
	if data.Crosslink.StartEpoch != parentCrosslink.EndEpoch {
		return nil, fmt.Errorf("expected crosslink start epoch %d, received %d",
			parentCrosslink.EndEpoch, data.Crosslink.StartEpoch)
	}
	if data.Crosslink.EndEpoch != endEpoch {
		return nil, fmt.Errorf("expected crosslink end epoch %d, received %d",
			endEpoch, data.Crosslink.EndEpoch)
	}
	crosslinkParentRoot, err := ssz.HashTreeRoot(parentCrosslink)
	if err != nil {
		return nil, fmt.Errorf("could not tree hash parent crosslink: %v", err)
	}
	if !bytes.Equal(data.Crosslink.ParentRoot, crosslinkParentRoot[:]) {
		return nil, fmt.Errorf(
			"mismatched parent crosslink root, expected %#x, received %#x",
			crosslinkParentRoot,
			data.Crosslink.ParentRoot,
		)
	}
	// To be removed in Phase 1
	if !bytes.Equal(data.Crosslink.DataRoot, params.BeaconConfig().ZeroHash[:]) {
		return nil, fmt.Errorf("expected data root %#x == ZERO_HASH", data.Crosslink.DataRoot)
	}
	indexedAtt, err := ConvertToIndexed(beaconState, att)
	if err != nil {
		return nil, fmt.Errorf("could not convert to indexed attestation: %v", err)
	}
	if err := VerifyIndexedAttestation(indexedAtt, verifySignatures); err != nil {
		return nil, fmt.Errorf("could not verify indexed attestation: %v", err)
	}
	return beaconState, nil
}

// ConvertToIndexed converts attestation to (almost) indexed-verifiable form.
//
// Spec pseudocode definition:
//   def convert_to_indexed(state: BeaconState, attestation: Attestation) -> IndexedAttestation:
//    """
//    Convert ``attestation`` to (almost) indexed-verifiable form.
//    """
//    attesting_indices = get_attesting_indices(state, attestation.data, attestation.aggregation_bitfield)
//    custody_bit_1_indices = get_attesting_indices(state, attestation.data, attestation.custody_bitfield)
//    assert custody_bit_1_indices.issubset(attesting_indices)
//    custody_bit_0_indices = attesting_indices.difference(custody_bit_1_indices)
//
//    return IndexedAttestation(
//        custody_bit_0_indices=sorted(custody_bit_0_indices),
//        custody_bit_1_indices=sorted(custody_bit_1_indices),
//        data=attestation.data,
//        signature=attestation.signature,
//    )
func ConvertToIndexed(state *pb.BeaconState, attestation *pb.Attestation) (*pb.IndexedAttestation, error) {
	attIndices, err := helpers.AttestingIndices(state, attestation.Data, attestation.AggregationBits)
	if err != nil {
		return nil, fmt.Errorf("could not get attesting indices: %v", err)
	}
	cb1i, _ := helpers.AttestingIndices(state, attestation.Data, attestation.CustodyBits)
	if !sliceutil.SubsetUint64(cb1i, attIndices) {
		return nil, fmt.Errorf("%v is not a subset of %v", cb1i, attIndices)
	}
	cb1Map := make(map[uint64]bool)
	for _, idx := range cb1i {
		cb1Map[idx] = true
	}
	cb0i := []uint64{}
	for _, idx := range attIndices {
		if !cb1Map[idx] {
			cb0i = append(cb0i, idx)
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
func VerifyIndexedAttestation(indexedAtt *pb.IndexedAttestation, verifySignatures bool) error {
	custodyBit0Indices := indexedAtt.CustodyBit_0Indices
	custodyBit1Indices := indexedAtt.CustodyBit_1Indices

	// To be removed in phase 1
	if len(custodyBit1Indices) > 0 {
		return fmt.Errorf("expected no bit 1 indices, received %v", len(custodyBit1Indices))
	}

	maxIndices := params.BeaconConfig().MaxIndicesPerAttestation
	totalIndicesLength := uint64(len(custodyBit0Indices) + len(custodyBit1Indices))
	if maxIndices < totalIndicesLength || totalIndicesLength < 1 {
		return fmt.Errorf("over max number of allowed indices per attestation: %d", totalIndicesLength)
	}
	custodyBitIntersection := sliceutil.IntersectionUint64(custodyBit0Indices, custodyBit1Indices)
	if len(custodyBitIntersection) != 0 {
		return fmt.Errorf("expected disjoint indices intersection, received %v", custodyBitIntersection)
	}

	if verifySignatures {
		// TODO(#2307): Update using BLS signature verification.
	}
	return nil
}

// ProcessDeposits is one of the operations performed on each processed
// beacon block to verify queued validators from the Ethereum 1.0 Deposit Contract
// into the beacon chain.
//
// Spec pseudocode definition:
//   For each deposit in block.body.deposits:
//     process_deposit(state, deposit)
func ProcessDeposits(
	beaconState *pb.BeaconState,
	body *pb.BeaconBlockBody,
	verifySignatures bool,
) (*pb.BeaconState, error) {
	var err error
	deposits := body.Deposits

	valIndexMap := stateutils.ValidatorIndexMap(beaconState)
	for _, deposit := range deposits {
		beaconState, err = ProcessDeposit(beaconState, deposit, valIndexMap, verifySignatures, true)
		if err != nil {
			return nil, fmt.Errorf("could not process deposit from %#x: %v", bytesutil.Trunc(deposit.Data.Pubkey), err)
		}
	}
	return beaconState, nil
}

// ProcessDeposit takes in a deposit object and inserts it
// into the registry as a new validator or balance change.
//
// Spec pseudocode definition:
//   def process_deposit(state: BeaconState, deposit: Deposit) -> None:
//     """
//     Process an Eth1 deposit, registering a validator or increasing its balance.
//     """
//     # Verify the Merkle branch
//     assert verify_merkle_branch(
//         leaf=hash_tree_root(deposit.data),
//         proof=deposit.proof,
//         depth=DEPOSIT_CONTRACT_TREE_DEPTH,
//         index=deposit.index,
//         root=state.latest_eth1_data.deposit_root,
//     )
//
//     # Deposits must be processed in order
//     assert deposit.index == state.deposit_index
//     state.deposit_index += 1
//
//     pubkey = deposit.data.pubkey
//     amount = deposit.data.amount
//     validator_pubkeys = [v.pubkey for v in state.validator_registry]
//     if pubkey not in validator_pubkeys:
//         # Verify the deposit signature (proof of possession).
//         # Invalid signatures are allowed by the deposit contract, and hence included on-chain, but must not be processed.
//         if not bls_verify(pubkey, signing_root(deposit.data), deposit.data.signature, get_domain(state, DOMAIN_DEPOSIT)):
//             return
//
//         # Add validator and balance entries
//         state.validator_registry.append(Validator(
//             pubkey=pubkey,
//             withdrawal_credentials=deposit.data.withdrawal_credentials,
//             activation_eligibility_epoch=FAR_FUTURE_EPOCH,
//             activation_epoch=FAR_FUTURE_EPOCH,
//             exit_epoch=FAR_FUTURE_EPOCH,
//             withdrawable_epoch=FAR_FUTURE_EPOCH,
//             effective_balance=min(amount - amount % EFFECTIVE_BALANCE_INCREMENT, MAX_EFFECTIVE_BALANCE)
//         ))
//         state.balances.append(amount)
//     else:
//         # Increase balance by deposit amount
//         index = validator_pubkeys.index(pubkey)
//         increase_balance(state, index, amount)
func ProcessDeposit(
	beaconState *pb.BeaconState,
	deposit *pb.Deposit,
	valIndexMap map[[32]byte]int,
	verifySignatures bool,
	verifyTree bool,
) (*pb.BeaconState, error) {
	if err := verifyDeposit(beaconState, deposit, verifyTree); err != nil {
		return nil, fmt.Errorf("could not verify deposit from #%x: %v", bytesutil.Trunc(deposit.Data.Pubkey), err)
	}
	beaconState.Eth1DepositIndex++
	pubKey := deposit.Data.Pubkey
	amount := deposit.Data.Amount
	index, ok := valIndexMap[bytesutil.ToBytes32(pubKey)]
	if !ok {
		if verifySignatures {
			// TODO(#2307): Use BLS verification of proof of possession.
		}
		effectiveBalance := amount - (amount % params.BeaconConfig().EffectiveBalanceIncrement)
		if params.BeaconConfig().MaxEffectiveBalance < effectiveBalance {
			effectiveBalance = params.BeaconConfig().MaxEffectiveBalance
		}
		beaconState.Validators = append(beaconState.Validators, &pb.Validator{
			Pubkey:                     pubKey,
			WithdrawalCredentials:      deposit.Data.WithdrawalCredentials,
			ActivationEligibilityEpoch: params.BeaconConfig().FarFutureEpoch,
			ActivationEpoch:            params.BeaconConfig().FarFutureEpoch,
			ExitEpoch:                  params.BeaconConfig().FarFutureEpoch,
			WithdrawableEpoch:          params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance:           effectiveBalance,
		})
		beaconState.Balances = append(beaconState.Balances, amount)
	} else {
		beaconState = helpers.IncreaseBalance(beaconState, uint64(index), amount)
	}

	return beaconState, nil
}

func verifyDeposit(beaconState *pb.BeaconState, deposit *pb.Deposit, verifyTree bool) error {
	if verifyTree {
		// Verify Merkle proof of deposit and deposit trie root.
		receiptRoot := beaconState.Eth1Data.DepositRoot
		leaf, err := ssz.HashTreeRoot(deposit.Data)
		if err != nil {
			return fmt.Errorf("could not tree hash deposit data: %v", err)
		}
		if ok := trieutil.VerifyMerkleProof(
			receiptRoot,
			leaf[:],
			int(beaconState.Eth1DepositIndex),
			deposit.Proof,
		); !ok {
			return fmt.Errorf(
				"deposit merkle branch of deposit root did not verify for root: %#x",
				receiptRoot,
			)
		}
	}

	return nil
}

// ProcessVolundaryExits is one of the operations performed
// on each processed beacon block to determine which validators
// should exit the state's validator registry.
//
// Spec pseudocode definition:
//   def process_voluntary_exit(state: BeaconState, exit: VoluntaryExit) -> None:
//    """
//    Process ``VoluntaryExit`` operation.
//    """
//    validator = state.validator_registry[exit.validator_index]
//    # Verify the validator is active
//    assert is_active_validator(validator, get_current_epoch(state))
//    # Verify the validator has not yet exited
//    assert validator.exit_epoch == FAR_FUTURE_EPOCH
//    # Exits must specify an epoch when they become valid; they are not valid before then
//    assert get_current_epoch(state) >= exit.epoch
//    # Verify the validator has been active long enough
//    assert get_current_epoch(state) >= validator.activation_epoch + PERSISTENT_COMMITTEE_PERIOD
//    # Verify signature
//    domain = get_domain(state, DOMAIN_VOLUNTARY_EXIT, exit.epoch)
//    assert bls_verify(validator.pubkey, signing_root(exit), exit.signature, domain)
//    # Initiate exit
//    initiate_validator_exit(state, exit.validator_index)
func ProcessVolundaryExits(
	beaconState *pb.BeaconState,
	body *pb.BeaconBlockBody,
	verifySignatures bool,
) (*pb.BeaconState, error) {
	var err error
	exits := body.VoluntaryExits

	for idx, exit := range exits {
		if err := verifyExit(beaconState, exit, verifySignatures); err != nil {
			return nil, fmt.Errorf("could not verify exit #%d: %v", idx, err)
		}
		beaconState, err = v.InitiateValidatorExit(beaconState, exit.ValidatorIndex)
		if err != nil {
			return nil, err
		}
	}
	return beaconState, nil
}

func verifyExit(beaconState *pb.BeaconState, exit *pb.VoluntaryExit, verifySignatures bool) error {
	validator := beaconState.Validators[exit.ValidatorIndex]
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

// ProcessTransfers is one of the operations performed
// on each processed beacon block to determine transfers between beacon chain balances.
//
// Spec pseudocode definition:
//   def process_transfer(state: BeaconState, transfer: Transfer) -> None:
//    """
//    Process ``Transfer`` operation.
//    """
//    # Verify the amount and fee are not individually too big (for anti-overflow purposes)
//    assert state.balances[transfer.sender] >= max(transfer.amount, transfer.fee)
//    # A transfer is valid in only one slot
//    assert state.slot == transfer.slot
//    # Sender must be not yet eligible for activation, withdrawn, or transfer balance over MAX_EFFECTIVE_BALANCE
//    assert (
//        state.validator_registry[transfer.sender].activation_eligibility_epoch == FAR_FUTURE_EPOCH or
//        get_current_epoch(state) >= state.validator_registry[transfer.sender].withdrawable_epoch or
//        transfer.amount + transfer.fee + MAX_EFFECTIVE_BALANCE <= state.balances[transfer.sender]
//    )
//    # Verify that the pubkey is valid
//    assert (
//        state.validator_registry[transfer.sender].withdrawal_credentials ==
//        int_to_bytes(BLS_WITHDRAWAL_PREFIX, length=1) + hash(transfer.pubkey)[1:]
//    )
//    # Verify that the signature is valid
//    assert bls_verify(transfer.pubkey, signing_root(transfer), transfer.signature, get_domain(state, DOMAIN_TRANSFER))
//    # Process the transfer
//    decrease_balance(state, transfer.sender, transfer.amount + transfer.fee)
//    increase_balance(state, transfer.recipient, transfer.amount)
//    increase_balance(state, get_beacon_proposer_index(state), transfer.fee)
//    # Verify balances are not dust
//    assert not (0 < state.balances[transfer.sender] < MIN_DEPOSIT_AMOUNT)
//    assert not (0 < state.balances[transfer.recipient] < MIN_DEPOSIT_AMOUNT)
func ProcessTransfers(
	beaconState *pb.BeaconState,
	body *pb.BeaconBlockBody,
	verifySignatures bool,
) (*pb.BeaconState, error) {
	transfers := body.Transfers

	for idx, transfer := range transfers {
		if err := verifyTransfer(beaconState, transfer, verifySignatures); err != nil {
			return nil, fmt.Errorf("could not verify transfer %d: %v", idx, err)
		}
		// Process the transfer between accounts.
		beaconState = helpers.DecreaseBalance(beaconState, transfer.Sender, transfer.Amount+transfer.Fee)
		beaconState = helpers.IncreaseBalance(beaconState, transfer.Recipient, transfer.Amount)
		proposerIndex, err := helpers.BeaconProposerIndex(beaconState)
		if err != nil {
			return nil, fmt.Errorf("could not determine beacon proposer index: %v", err)
		}
		beaconState = helpers.IncreaseBalance(beaconState, proposerIndex, transfer.Fee)

		// Finally, we verify balances will not go below the mininum.
		if beaconState.Balances[transfer.Sender] < params.BeaconConfig().MinDepositAmount &&
			0 < beaconState.Balances[transfer.Sender] {
			return nil, fmt.Errorf(
				"sender balance below critical level: %v",
				beaconState.Balances[transfer.Sender],
			)
		}
		if beaconState.Balances[transfer.Recipient] < params.BeaconConfig().MinDepositAmount &&
			0 < beaconState.Balances[transfer.Recipient] {
			return nil, fmt.Errorf(
				"recipient balance below critical level: %v",
				beaconState.Balances[transfer.Recipient],
			)
		}
	}
	return beaconState, nil
}

func verifyTransfer(beaconState *pb.BeaconState, transfer *pb.Transfer, verifySignatures bool) error {
	maxVal := transfer.Fee
	if transfer.Amount > maxVal {
		maxVal = transfer.Amount
	}
	sender := beaconState.Validators[transfer.Sender]
	senderBalance := beaconState.Balances[transfer.Sender]
	// Verify the amount and fee are not individually too big (for anti-overflow purposes).
	if senderBalance < maxVal {
		return fmt.Errorf("expected sender balance %d >= %d", senderBalance, maxVal)
	}
	// A transfer is valid in only one slot.
	if beaconState.Slot != transfer.Slot {
		return fmt.Errorf("expected beacon state slot %d == transfer slot %d", beaconState.Slot, transfer.Slot)
	}

	// Sender must be not yet eligible for activation, withdrawn, or transfer balance over MAX_EFFECTIVE_BALANCE.
	senderNotActivationEligible := sender.ActivationEligibilityEpoch == params.BeaconConfig().FarFutureEpoch
	senderNotWithdrawn := helpers.CurrentEpoch(beaconState) >= sender.WithdrawableEpoch
	underMaxTransfer := transfer.Amount+transfer.Fee+params.BeaconConfig().MaxEffectiveBalance <= senderBalance

	if !(senderNotActivationEligible || senderNotWithdrawn || underMaxTransfer) {
		return fmt.Errorf(
			"expected activation eligiblity: false or withdrawn: false or over max transfer: false, received %v %v %v",
			senderNotActivationEligible,
			senderNotWithdrawn,
			underMaxTransfer,
		)
	}
	// Verify that the pubkey is valid.
	buf := []byte{params.BeaconConfig().BLSWithdrawalPrefixByte}
	hashed := hashutil.Hash(transfer.Pubkey)
	buf = append(buf, hashed[:][1:]...)
	if !bytes.Equal(sender.WithdrawalCredentials, buf) {
		return fmt.Errorf("invalid public key, expected %v, received %v", buf, sender.WithdrawalCredentials)
	}
	if verifySignatures {
		// TODO(#258): Integrate BLS signature verification for transfers.
	}
	return nil
}

// ClearEth1DataVoteCache clears the eth1 data vote count cache.
func ClearEth1DataVoteCache() {
	eth1DataCache = cache.NewEth1DataVoteCache()
}
