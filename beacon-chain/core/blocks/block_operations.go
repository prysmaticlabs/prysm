package blocks

import (
	"bytes"
	"encoding/binary"

	"fmt"
	"sort"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "blocks")

var eth1DataCache = cache.NewEth1DataVoteCache()

func verifySigningRoot(obj interface{}, pub []byte, signature []byte, domain uint64) error {
	publicKey, err := bls.PublicKeyFromBytes(pub)
	if err != nil {
		return errors.Wrap(err, "could not convert bytes to public key")
	}
	sig, err := bls.SignatureFromBytes(signature)
	if err != nil {
		return errors.Wrap(err, "could not convert bytes to signature")
	}
	root, err := ssz.SigningRoot(obj)
	if err != nil {
		return errors.Wrap(err, "could not get signing root")
	}
	if !sig.Verify(root[:], publicKey, domain) {
		return fmt.Errorf("signature did not verify")
	}
	return nil
}

func verifySignature(signedData []byte, pub []byte, signature []byte, domain uint64) error {
	publicKey, err := bls.PublicKeyFromBytes(pub)
	if err != nil {
		return errors.Wrap(err, "could not convert bytes to public key")
	}
	sig, err := bls.SignatureFromBytes(signature)
	if err != nil {
		return errors.Wrap(err, "could not convert bytes to signature")
	}
	if !sig.Verify(signedData, publicKey, domain) {
		return fmt.Errorf("signature did not verify")
	}
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
func ProcessEth1DataInBlock(beaconState *pb.BeaconState, block *ethpb.BeaconBlock) (*pb.BeaconState, error) {
	beaconState.Eth1DataVotes = append(beaconState.Eth1DataVotes, block.Body.Eth1Data)

	hasSupport, err := Eth1DataHasEnoughSupport(beaconState, block.Body.Eth1Data)
	if err != nil {
		return nil, err
	}

	if hasSupport {
		beaconState.Eth1Data = block.Body.Eth1Data
	}

	return beaconState, nil
}

// Eth1DataHasEnoughSupport returns true when the given eth1data has more than 50% votes in the
// eth1 voting period. A vote is cast by including eth1data in a block and part of state processing
// appends eth1data to the state in the Eth1DataVotes list. Iterating through this list checks the
// votes to see if they match the eth1data.
func Eth1DataHasEnoughSupport(beaconState *pb.BeaconState, data *ethpb.Eth1Data) (bool, error) {
	eth1DataHash, err := hashutil.HashProto(data)
	if err != nil {
		return false, errors.Wrap(err, "could not hash eth1data")
	}
	voteCount, err := eth1DataCache.Eth1DataVote(eth1DataHash)
	if err != nil {
		return false, errors.Wrap(err, "could not retrieve eth1 data vote cache")
	}

	if voteCount == 0 {
		for _, vote := range beaconState.Eth1DataVotes {
			if proto.Equal(vote, data) {
				voteCount++
			}
		}
	} else {
		voteCount++
	}

	if err := eth1DataCache.AddEth1DataVote(&cache.Eth1DataVote{
		Eth1DataHash: eth1DataHash,
		VoteCount:    voteCount,
	}); err != nil {
		return false, errors.Wrap(err, "could not save eth1 data vote cache")
	}

	// If 50+% majority converged on the same eth1data, then it has enough support to update the
	// state.
	return voteCount*2 > params.BeaconConfig().SlotsPerEth1VotingPeriod, nil
}

// ProcessBlockHeader validates a block by its header.
//
// Spec pseudocode definition:
//
//  def process_block_header(state: BeaconState, block: BeaconBlock) -> None:
//    # Verify that the slots match
//    assert block.slot == state.slot
//    # Verify that the parent matches
//    assert block.parent_root == signing_root(state.latest_block_header)
//    # Save current block as the new latest block
//    state.latest_block_header = BeaconBlockHeader(
//        slot=block.slot,
//        parent_root=block.parent_root,
//        # state_root: zeroed, overwritten in the next `process_slot` call
//        body_root=hash_tree_root(block.body),
//		  # signature is always zeroed
//    )
//    # Verify proposer is not slashed
//    proposer = state.validators[get_beacon_proposer_index(state)]
//    assert not proposer.slashed
//    # Verify proposer signature
//    assert bls_verify(proposer.pubkey, signing_root(block), block.signature, get_domain(state, DOMAIN_BEACON_PROPOSER))
func ProcessBlockHeader(
	beaconState *pb.BeaconState,
	block *ethpb.BeaconBlock,
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
	beaconState.LatestBlockHeader = &ethpb.BeaconBlockHeader{
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

	// Verify proposer signature.
	currentEpoch := helpers.CurrentEpoch(beaconState)
	domain := helpers.Domain(beaconState, currentEpoch, params.BeaconConfig().DomainBeaconProposer)
	if err := verifySigningRoot(block, proposer.PublicKey, block.Signature, domain); err != nil {
		return nil, errors.Wrap(err, "could not verify block signature")
	}

	return beaconState, nil
}

// ProcessBlockHeaderNoVerify validates a block by its header but skips proposer
// signature verification.
//
// // WARNING: This method does not verify proposer signature. This is used for proposer to compute state root
// using a unsigned block.
//
// Spec pseudocode definition:
//  def process_block_header(state: BeaconState, block: BeaconBlock) -> None:
//    # Verify that the slots match
//    assert block.slot == state.slot
//    # Verify that the parent matches
//    assert block.parent_root == signing_root(state.latest_block_header)
//    # Save current block as the new latest block
//    state.latest_block_header = BeaconBlockHeader(
//        slot=block.slot,
//        parent_root=block.parent_root,
//        # state_root: zeroed, overwritten in the next `process_slot` call
//        body_root=hash_tree_root(block.body),
//		  # signature is always zeroed
//    )
//    # Verify proposer is not slashed
//    proposer = state.validators[get_beacon_proposer_index(state)]
//    assert not proposer.slashed
func ProcessBlockHeaderNoVerify(
	beaconState *pb.BeaconState,
	block *ethpb.BeaconBlock,
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
	beaconState.LatestBlockHeader = &ethpb.BeaconBlockHeader{
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
	body *ethpb.BeaconBlockBody,
) (*pb.BeaconState, error) {
	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	if err != nil {
		return nil, errors.Wrap(err, "could not get beacon proposer index")
	}
	proposerPub := beaconState.Validators[proposerIdx].PublicKey

	currentEpoch := helpers.CurrentEpoch(beaconState)
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, currentEpoch)

	domain := helpers.Domain(beaconState, currentEpoch, params.BeaconConfig().DomainRandao)
	if err := verifySignature(buf, proposerPub, body.RandaoReveal, domain); err != nil {
		return nil, errors.Wrap(err, "could not verify block randao")
	}

	// If block randao passed verification, we XOR the state's latest randao mix with the block's
	// randao and update the state's corresponding latest randao mix value.
	latestMixesLength := params.BeaconConfig().EpochsPerHistoricalVector
	latestMixSlice := beaconState.RandaoMixes[currentEpoch%latestMixesLength]
	blockRandaoReveal := hashutil.Hash(body.RandaoReveal)
	for i, x := range blockRandaoReveal {
		latestMixSlice[i] ^= x
	}
	beaconState.RandaoMixes[currentEpoch%latestMixesLength] = latestMixSlice
	return beaconState, nil
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
	body *ethpb.BeaconBlockBody,
) (*pb.BeaconState, error) {
	var err error
	for idx, slashing := range body.ProposerSlashings {
		if int(slashing.ProposerIndex) >= len(beaconState.Validators) {
			return nil, fmt.Errorf("invalid proposer index given in slashing %d", slashing.ProposerIndex)
		}
		if err = VerifyProposerSlashing(beaconState, slashing); err != nil {
			return nil, errors.Wrapf(err, "could not verify proposer slashing %d", idx)
		}
		beaconState, err = v.SlashValidator(
			beaconState, slashing.ProposerIndex, 0, /* proposer is whistleblower */
		)
		if err != nil {
			return nil, errors.Wrapf(err, "could not slash proposer index %d", slashing.ProposerIndex)
		}
	}
	return beaconState, nil
}

// VerifyProposerSlashing verifies that the data provided fro slashing is valid.
func VerifyProposerSlashing(
	beaconState *pb.BeaconState,
	slashing *ethpb.ProposerSlashing,
) error {
	headerEpoch1 := helpers.SlotToEpoch(slashing.Header_1.Slot)
	headerEpoch2 := helpers.SlotToEpoch(slashing.Header_2.Slot)
	proposer := beaconState.Validators[slashing.ProposerIndex]

	if headerEpoch1 != headerEpoch2 {
		return fmt.Errorf("mismatched header epochs, received %d == %d", headerEpoch1, headerEpoch2)
	}
	if proto.Equal(slashing.Header_1, slashing.Header_2) {
		return errors.New("expected slashing headers to differ")
	}
	if !helpers.IsSlashableValidator(proposer, helpers.CurrentEpoch(beaconState)) {
		return fmt.Errorf("validator with key %#x is not slashable", proposer.PublicKey)
	}
	// Using headerEpoch1 here because both of the headers should have the same epoch.
	domain := helpers.Domain(beaconState, headerEpoch1, params.BeaconConfig().DomainBeaconProposer)
	headers := append([]*ethpb.BeaconBlockHeader{slashing.Header_1}, slashing.Header_2)
	for _, header := range headers {
		if err := verifySigningRoot(header, proposer.PublicKey, header.Signature, domain); err != nil {
			return errors.Wrap(err, "could not verify beacon block header")
		}
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
//        if is_slashable_validator(state.validators[index], get_current_epoch(state)):
//            slash_validator(state, index)
//            slashed_any = True
//    assert slashed_any
func ProcessAttesterSlashings(
	beaconState *pb.BeaconState,
	body *ethpb.BeaconBlockBody,
) (*pb.BeaconState, error) {
	for idx, slashing := range body.AttesterSlashings {
		if err := VerifyAttesterSlashing(beaconState, slashing); err != nil {
			return nil, errors.Wrapf(err, "could not verify attester slashing %d", idx)
		}
		slashableIndices := slashableAttesterIndices(slashing)
		sort.SliceStable(slashableIndices, func(i, j int) bool {
			return slashableIndices[i] < slashableIndices[j]
		})
		currentEpoch := helpers.CurrentEpoch(beaconState)
		var err error
		var slashedAny bool
		for _, validatorIndex := range slashableIndices {
			if helpers.IsSlashableValidator(beaconState.Validators[validatorIndex], currentEpoch) {
				beaconState, err = v.SlashValidator(beaconState, validatorIndex, 0)
				if err != nil {
					return nil, errors.Wrapf(err, "could not slash validator index %d",
						validatorIndex)
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

// VerifyAttesterSlashing validates the attestation data in both attestations in the slashing object.
func VerifyAttesterSlashing(beaconState *pb.BeaconState, slashing *ethpb.AttesterSlashing) error {
	att1 := slashing.Attestation_1
	att2 := slashing.Attestation_2
	data1 := att1.Data
	data2 := att2.Data
	if !IsSlashableAttestationData(data1, data2) {
		return errors.New("attestations are not slashable")
	}
	if err := VerifyIndexedAttestation(beaconState, att1); err != nil {
		return errors.Wrap(err, "could not validate indexed attestation")
	}
	if err := VerifyIndexedAttestation(beaconState, att2); err != nil {
		return errors.Wrap(err, "could not validate indexed attestation")
	}
	return nil
}

// IsSlashableAttestationData verifies a slashing against the Casper Proof of Stake FFG rules.
//
// Spec pseudocode definition:
//   def is_slashable_attestation_data(data_1: AttestationData, data_2: AttestationData) -> bool:
//    """
//    Check if ``data_1`` and ``data_2`` are slashable according to Casper FFG rules.
//    """
//    return (
//        # Double vote
//        (data_1 != data_2 and data_1.target.epoch == data_2.target.epoch) or
//        # Surround vote
//        (data_1.source.epoch < data_2.source.epoch and data_2.target.epoch < data_1.target.epoch)
//    )
func IsSlashableAttestationData(data1 *ethpb.AttestationData, data2 *ethpb.AttestationData) bool {
	isDoubleVote := !proto.Equal(data1, data2) && data1.Target.Epoch == data2.Target.Epoch
	isSurroundVote := data1.Source.Epoch < data2.Source.Epoch && data2.Target.Epoch < data1.Target.Epoch
	return isDoubleVote || isSurroundVote
}

func slashableAttesterIndices(slashing *ethpb.AttesterSlashing) []uint64 {
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
	body *ethpb.BeaconBlockBody,
) (*pb.BeaconState, error) {
	var err error
	for idx, attestation := range body.Attestations {
		beaconState, err = ProcessAttestation(beaconState, attestation)
		if err != nil {
			return nil, errors.Wrapf(err, "could not verify attestation at index %d in block", idx)
		}
	}
	return beaconState, nil
}

// ProcessAttestationsNoVerify applies processing operations to a block's inner attestation
// records. The only difference would be that the attestation signature would not be verified.
func ProcessAttestationsNoVerify(
	beaconState *pb.BeaconState,
	body *ethpb.BeaconBlockBody,
) (*pb.BeaconState, error) {
	var err error
	for idx, attestation := range body.Attestations {
		beaconState, err = ProcessAttestationNoVerify(beaconState, attestation)
		if err != nil {
			return nil, errors.Wrapf(err, "could not verify attestation at index %d in block", idx)
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
//    assert data.crosslink.shard < SHARD_COUNT
//    assert data.target.epoch in (get_previous_epoch(state), get_current_epoch(state))
//
//    attestation_slot = get_attestation_data_slot(state, data)
//    assert attestation_slot + MIN_ATTESTATION_INCLUSION_DELAY <= state.slot <= attestation_slot + SLOTS_PER_EPOCH
//
//    committee = get_crosslink_committee(state, data.target.epoch, data.crosslink.shard)
//    assert len(attestation.aggregation_bits) == len(attestation.custody_bits) == len(committee)
//
//    pending_attestation = PendingAttestation(
//        data=data,
//        aggregation_bitfield=attestation.aggregation_bitfield,
//        inclusion_delay=state.slot - attestation_slot,
//        proposer_index=get_beacon_proposer_index(state),
//    )
//
//    if data.target_epoch == get_current_epoch(state):
//      assert data.source == state.current_justified_checkpoint
//      parent_crosslink = state.current_crosslinks[data.crosslink.shard]
//      state.current_epoch_attestations.append(pending_attestation)
//    else:
//      assert data.source == state.previous_justified_checkpoint
//      parent_crosslink = state.previous_crosslinks[data.crosslink.shard]
//      state.previous_epoch_attestations.append(pending_attestation)
//
//    # Check crosslink against expected parent crosslink
//    assert data.crosslink.parent_root == hash_tree_root(parent_crosslink)
//    assert data.crosslink.start_epoch == parent_crosslink.end_epoch
//    assert data.crosslink.end_epoch == min(data.target.epoch, parent_crosslink.end_epoch + MAX_EPOCHS_PER_CROSSLINK)
//    assert data.crosslink.data_root == Bytes32()  # [to be removed in phase 1]
//
//    # Check signature
//    assert is_valid_indexed_attestation(state, get_indexed_attestation(state, attestation))
func ProcessAttestation(beaconState *pb.BeaconState, att *ethpb.Attestation) (*pb.BeaconState, error) {
	beaconState, err := ProcessAttestationNoVerify(beaconState, att)
	if err != nil {
		return nil, err
	}
	return beaconState, VerifyAttestation(beaconState, att)
}

// ProcessAttestationNoVerify processes the attestation without verifying the attestation signature. This
// method is used to validate attestations whose signatures have already been verified.
func ProcessAttestationNoVerify(beaconState *pb.BeaconState, att *ethpb.Attestation) (*pb.BeaconState, error) {
	data := att.Data

	if data.Crosslink.Shard > params.BeaconConfig().ShardCount {
		return nil, fmt.Errorf(
			"expected crosslink shard %d to be less than SHARD_COUNT %d",
			data.Crosslink.Shard,
			params.BeaconConfig().ShardCount,
		)
	}

	if data.Target.Epoch != helpers.PrevEpoch(beaconState) && data.Target.Epoch != helpers.CurrentEpoch(beaconState) {
		return nil, fmt.Errorf(
			"expected target epoch (%d) to be the previous epoch (%d) or the current epoch (%d)",
			data.Target.Epoch,
			helpers.PrevEpoch(beaconState),
			helpers.CurrentEpoch(beaconState),
		)
	}

	attestationSlot, err := helpers.AttestationDataSlot(beaconState, data)
	if err != nil {
		return nil, errors.Wrap(err, "could not get attestation slot")
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

	if err := helpers.VerifyAttestationBitfieldLengths(beaconState, att); err != nil {
		return nil, errors.Wrap(err, "could not verify attestation bitfields")
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

	var ffgSourceEpoch uint64
	var ffgSourceRoot []byte
	var ffgTargetEpoch uint64
	var parentCrosslink *ethpb.Crosslink
	if data.Target.Epoch == helpers.CurrentEpoch(beaconState) {
		ffgSourceEpoch = beaconState.CurrentJustifiedCheckpoint.Epoch
		ffgSourceRoot = beaconState.CurrentJustifiedCheckpoint.Root
		ffgTargetEpoch = helpers.CurrentEpoch(beaconState)
		crosslinkShard := data.Crosslink.Shard
		if int(crosslinkShard) >= len(beaconState.CurrentCrosslinks) {
			return nil, fmt.Errorf("invalid shard given in attestation: %d", crosslinkShard)
		}

		parentCrosslink = beaconState.CurrentCrosslinks[crosslinkShard]
		beaconState.CurrentEpochAttestations = append(beaconState.CurrentEpochAttestations, pendingAtt)
	} else {
		ffgSourceEpoch = beaconState.PreviousJustifiedCheckpoint.Epoch
		ffgSourceRoot = beaconState.PreviousJustifiedCheckpoint.Root
		ffgTargetEpoch = helpers.PrevEpoch(beaconState)
		crosslinkShard := data.Crosslink.Shard
		if int(crosslinkShard) >= len(beaconState.PreviousCrosslinks) {
			return nil, fmt.Errorf("invalid shard given in attestation: %d", crosslinkShard)
		}
		parentCrosslink = beaconState.PreviousCrosslinks[crosslinkShard]
		beaconState.PreviousEpochAttestations = append(beaconState.PreviousEpochAttestations, pendingAtt)
	}
	if data.Source.Epoch != ffgSourceEpoch {
		return nil, fmt.Errorf("expected source epoch %d, received %d", ffgSourceEpoch, data.Source.Epoch)
	}
	if !bytes.Equal(data.Source.Root, ffgSourceRoot) {
		return nil, fmt.Errorf("expected source root %#x, received %#x", ffgSourceRoot, data.Source.Root)
	}
	if data.Target.Epoch != ffgTargetEpoch {
		return nil, fmt.Errorf("expected target epoch %d, received %d", ffgTargetEpoch, data.Target.Epoch)
	}
	endEpoch := parentCrosslink.EndEpoch + params.BeaconConfig().MaxEpochsPerCrosslink
	if data.Target.Epoch < endEpoch {
		endEpoch = data.Target.Epoch
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
		return nil, errors.Wrap(err, "could not tree hash parent crosslink")
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
	return beaconState, nil
}

// ConvertToIndexed converts attestation to (almost) indexed-verifiable form.
//
// Spec pseudocode definition:
//   def get_indexed_attestation(state: BeaconState, attestation: Attestation) -> IndexedAttestation:
//    """
//    Return the indexed attestation corresponding to ``attestation``.
//    """
//    attesting_indices = get_attesting_indices(state, attestation.data, attestation.aggregation_bits)
//    custody_bit_1_indices = get_attesting_indices(state, attestation.data, attestation.custody_bits)
//    assert custody_bit_1_indices.issubset(attesting_indices)
//    custody_bit_0_indices = attesting_indices.difference(custody_bit_1_indices)
//
//    return IndexedAttestation(
//        custody_bit_0_indices=sorted(custody_bit_0_indices),
//        custody_bit_1_indices=sorted(custody_bit_1_indices),
//        data=attestation.data,
//        signature=attestation.signature,
//    )
func ConvertToIndexed(state *pb.BeaconState, attestation *ethpb.Attestation) (*ethpb.IndexedAttestation, error) {
	attIndices, err := helpers.AttestingIndices(state, attestation.Data, attestation.AggregationBits)
	if err != nil {
		return nil, errors.Wrap(err, "could not get attesting indices")
	}
	cb1i, err := helpers.AttestingIndices(state, attestation.Data, attestation.CustodyBits)
	if err != nil {
		return nil, err
	}
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
	sort.Slice(cb0i, func(i, j int) bool {
		return cb0i[i] < cb0i[j]
	})

	sort.Slice(cb1i, func(i, j int) bool {
		return cb1i[i] < cb1i[j]
	})
	inAtt := &ethpb.IndexedAttestation{
		Data:                attestation.Data,
		Signature:           attestation.Signature,
		CustodyBit_0Indices: cb0i,
		CustodyBit_1Indices: cb1i,
	}
	return inAtt, nil
}

// VerifyIndexedAttestation determines the validity of an indexed attestation.
//
// Spec pseudocode definition:
//  def is_valid_indexed_attestation(state: BeaconState, indexed_attestation: IndexedAttestation) -> bool:
//    """
//    Check if ``indexed_attestation`` has valid indices and signature.
//    """
//    bit_0_indices = indexed_attestation.custody_bit_0_indices
//    bit_1_indices = indexed_attestation.custody_bit_1_indices
//
//    # Verify no index has custody bit equal to 1 [to be removed in phase 1]
//    if not len(bit_1_indices) == 0:
//        return False
//    # Verify max number of indices
//    if not len(bit_0_indices) + len(bit_1_indices) <= MAX_VALIDATORS_PER_COMMITTEE:
//        return False
//    # Verify index sets are disjoint
//    if not len(set(bit_0_indices).intersection(bit_1_indices)) == 0:
//        return False
//    # Verify indices are sorted
//    if not (bit_0_indices == sorted(bit_0_indices) and bit_1_indices == sorted(bit_1_indices)):
//        return False
//    # Verify aggregate signature
//    if not bls_verify_multiple(
//        pubkeys=[
//            bls_aggregate_pubkeys([state.validators[i].pubkey for i in bit_0_indices]),
//            bls_aggregate_pubkeys([state.validators[i].pubkey for i in bit_1_indices]),
//        ],
//        message_hashes=[
//            hash_tree_root(AttestationDataAndCustodyBit(data=indexed_attestation.data, custody_bit=0b0)),
//            hash_tree_root(AttestationDataAndCustodyBit(data=indexed_attestation.data, custody_bit=0b1)),
//        ],
//        signature=indexed_attestation.signature,
//        domain=get_domain(state, DOMAIN_ATTESTATION, indexed_attestation.data.target.epoch),
//    ):
//        return False
//    return True
func VerifyIndexedAttestation(beaconState *pb.BeaconState, indexedAtt *ethpb.IndexedAttestation) error {
	custodyBit0Indices := indexedAtt.CustodyBit_0Indices
	custodyBit1Indices := indexedAtt.CustodyBit_1Indices

	// To be removed in phase 1
	if len(custodyBit1Indices) != 0 {
		return fmt.Errorf("expected no bit 1 indices, received %v", len(custodyBit1Indices))
	}

	maxIndices := params.BeaconConfig().MaxValidatorsPerCommittee
	totalIndicesLength := uint64(len(custodyBit0Indices) + len(custodyBit1Indices))
	if totalIndicesLength > maxIndices {
		return fmt.Errorf("over max number of allowed indices per attestation: %d", totalIndicesLength)
	}
	custodyBitIntersection := sliceutil.IntersectionUint64(custodyBit0Indices, custodyBit1Indices)
	if len(custodyBitIntersection) != 0 {
		return fmt.Errorf("expected disjoint indices intersection, received %v", custodyBitIntersection)
	}

	custodyBit0IndicesIsSorted := sort.SliceIsSorted(custodyBit0Indices, func(i, j int) bool {
		return custodyBit0Indices[i] < custodyBit0Indices[j]
	})

	if !custodyBit0IndicesIsSorted {
		return fmt.Errorf("custody Bit0 indices are not sorted, got %v", custodyBit0Indices)
	}

	custodyBit1IndicesIsSorted := sort.SliceIsSorted(custodyBit1Indices, func(i, j int) bool {
		return custodyBit1Indices[i] < custodyBit1Indices[j]
	})

	if !custodyBit1IndicesIsSorted {
		return fmt.Errorf("custody Bit1 indices are not sorted, got %v", custodyBit1Indices)
	}

	domain := helpers.Domain(beaconState, indexedAtt.Data.Target.Epoch, params.BeaconConfig().DomainAttestation)
	var pubkeys []*bls.PublicKey
	if len(custodyBit0Indices) > 0 {
		pubkey, err := bls.PublicKeyFromBytes(beaconState.Validators[custodyBit0Indices[0]].PublicKey)
		if err != nil {
			return errors.Wrap(err, "could not deserialize validator public key")
		}
		for _, i := range custodyBit0Indices[1:] {
			pk, err := bls.PublicKeyFromBytes(beaconState.Validators[i].PublicKey)
			if err != nil {
				return errors.Wrap(err, "could not deserialize validator public key")
			}
			pubkey.Aggregate(pk)
		}
		pubkeys = append(pubkeys, pubkey)
	}
	if len(custodyBit1Indices) > 0 {
		pubkey, err := bls.PublicKeyFromBytes(beaconState.Validators[custodyBit1Indices[0]].PublicKey)
		if err != nil {
			return errors.Wrap(err, "could not deserialize validator public key")
		}
		for _, i := range custodyBit1Indices[1:] {
			pk, err := bls.PublicKeyFromBytes(beaconState.Validators[i].PublicKey)
			if err != nil {
				return errors.Wrap(err, "could not deserialize validator public key")
			}
			pubkey.Aggregate(pk)
		}
		pubkeys = append(pubkeys, pubkey)
	}

	var msgs [][32]byte
	cus0 := &pb.AttestationDataAndCustodyBit{Data: indexedAtt.Data, CustodyBit: false}
	cus1 := &pb.AttestationDataAndCustodyBit{Data: indexedAtt.Data, CustodyBit: true}
	if len(custodyBit0Indices) > 0 {
		cus0Root, err := ssz.HashTreeRoot(cus0)
		if err != nil {
			return errors.Wrap(err, "could not tree hash att data and custody bit 0")
		}
		msgs = append(msgs, cus0Root)
	}
	if len(custodyBit1Indices) > 0 {
		cus1Root, err := ssz.HashTreeRoot(cus1)
		if err != nil {
			return errors.Wrap(err, "could not tree hash att data and custody bit 1")
		}
		msgs = append(msgs, cus1Root)
	}

	sig, err := bls.SignatureFromBytes(indexedAtt.Signature)
	if err != nil {
		return errors.Wrap(err, "could not convert bytes to signature")
	}

	hasVotes := len(custodyBit0Indices) > 0 || len(custodyBit1Indices) > 0

	if hasVotes && !sig.VerifyAggregate(pubkeys, msgs, domain) {
		return fmt.Errorf("attestation aggregation signature did not verify")
	}
	return nil
}

// VerifyAttestation converts and attestation into an indexed attestation and verifies
// the signature in that attestation.
func VerifyAttestation(beaconState *pb.BeaconState, att *ethpb.Attestation) error {
	indexedAtt, err := ConvertToIndexed(beaconState, att)
	if err != nil {
		return errors.Wrap(err, "could not convert to indexed attestation")
	}
	return VerifyIndexedAttestation(beaconState, indexedAtt)
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
	body *ethpb.BeaconBlockBody,
) (*pb.BeaconState, error) {
	var err error
	deposits := body.Deposits

	valIndexMap := stateutils.ValidatorIndexMap(beaconState)
	for _, deposit := range deposits {
		beaconState, err = ProcessDeposit(beaconState, deposit, valIndexMap)
		if err != nil {
			return nil, errors.Wrapf(err, "could not process deposit from %#x", bytesutil.Trunc(deposit.Data.PublicKey))
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
//         if not bls_verify(pubkey, signing_root(deposit.data), deposit.data.signature%d, get_domain(state, DOMAIN_DEPOSIT)):
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
func ProcessDeposit(beaconState *pb.BeaconState, deposit *ethpb.Deposit, valIndexMap map[[32]byte]int) (*pb.BeaconState, error) {
	if err := verifyDeposit(beaconState, deposit); err != nil {
		return nil, errors.Wrapf(err, "could not verify deposit from %#x", bytesutil.Trunc(deposit.Data.PublicKey))
	}
	beaconState.Eth1DepositIndex++
	pubKey := deposit.Data.PublicKey
	amount := deposit.Data.Amount
	index, ok := valIndexMap[bytesutil.ToBytes32(pubKey)]
	if !ok {
		domain := helpers.Domain(beaconState, helpers.CurrentEpoch(beaconState), params.BeaconConfig().DomainDeposit)
		depositSig := deposit.Data.Signature
		if err := verifySigningRoot(deposit.Data, pubKey, depositSig, domain); err != nil {
			// Ignore this error as in the spec pseudo code.
			log.Errorf("Skipping deposit: could not verify deposit data signature: %v", err)
			return beaconState, nil
		}

		effectiveBalance := amount - (amount % params.BeaconConfig().EffectiveBalanceIncrement)
		if params.BeaconConfig().MaxEffectiveBalance < effectiveBalance {
			effectiveBalance = params.BeaconConfig().MaxEffectiveBalance
		}
		beaconState.Validators = append(beaconState.Validators, &ethpb.Validator{
			PublicKey:                  pubKey,
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

func verifyDeposit(beaconState *pb.BeaconState, deposit *ethpb.Deposit) error {
	// Verify Merkle proof of deposit and deposit trie root.
	receiptRoot := beaconState.Eth1Data.DepositRoot
	leaf, err := ssz.HashTreeRoot(deposit.Data)
	if err != nil {
		return errors.Wrap(err, "could not tree hash deposit data")
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
	return nil
}

// ProcessVoluntaryExits is one of the operations performed
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
func ProcessVoluntaryExits(
	beaconState *pb.BeaconState,
	body *ethpb.BeaconBlockBody,
) (*pb.BeaconState, error) {
	var err error
	exits := body.VoluntaryExits

	for idx, exit := range exits {
		if err := VerifyExit(beaconState, exit); err != nil {
			return nil, errors.Wrapf(err, "could not verify exit %d", idx)
		}
		beaconState, err = v.InitiateValidatorExit(beaconState, exit.ValidatorIndex)
		if err != nil {
			return nil, err
		}
	}
	return beaconState, nil
}

// VerifyExit implements the spec defined validation for voluntary exits.
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
func VerifyExit(beaconState *pb.BeaconState, exit *ethpb.VoluntaryExit) error {
	if int(exit.ValidatorIndex) >= len(beaconState.Validators) {
		return fmt.Errorf("validator index out of bound %d > %d", exit.ValidatorIndex, len(beaconState.Validators))
	}

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
	domain := helpers.Domain(beaconState, exit.Epoch, params.BeaconConfig().DomainVoluntaryExit)
	if err := verifySigningRoot(exit, validator.PublicKey, exit.Signature, domain); err != nil {
		return errors.Wrap(err, "could not verify voluntary exit signature")
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
//    # Verify the balance the covers amount and fee (with overflow protection)
//	  assert state.balances[transfer.sender] >= max(transfer.amount + transfer.fee, transfer.amount, transfer.fee)
//    # A transfer is valid in only one slot
//    assert state.slot == transfer.slot
//    # SenderIndex must satisfy at least one of the following conditions in the parenthesis:
//    assert (
//		  # * Has not been activated
//        state.validator_registry[transfer.sender].activation_eligibility_epoch == FAR_FUTURE_EPOCH or
//        # * Is withdrawable
//        get_current_epoch(state) >= state.validator_registry[transfer.sender].withdrawable_epoch or
//        # * Balance after transfer is more than the effective balance threshold
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
	body *ethpb.BeaconBlockBody,
) (*pb.BeaconState, error) {
	transfers := body.Transfers

	for idx, transfer := range transfers {
		if err := verifyTransfer(beaconState, transfer); err != nil {
			return nil, errors.Wrapf(err, "could not verify transfer %d", idx)
		}
		// Process the transfer between accounts.
		beaconState = helpers.DecreaseBalance(beaconState, transfer.SenderIndex, transfer.Amount+transfer.Fee)
		beaconState = helpers.IncreaseBalance(beaconState, transfer.RecipientIndex, transfer.Amount)
		proposerIndex, err := helpers.BeaconProposerIndex(beaconState)
		if err != nil {
			return nil, errors.Wrap(err, "could not determine beacon proposer index")
		}
		beaconState = helpers.IncreaseBalance(beaconState, proposerIndex, transfer.Fee)

		// Finally, we verify balances will not go below the mininum.
		if beaconState.Balances[transfer.SenderIndex] < params.BeaconConfig().MinDepositAmount &&
			0 < beaconState.Balances[transfer.SenderIndex] {
			return nil, fmt.Errorf(
				"sender balance below critical level: %v",
				beaconState.Balances[transfer.SenderIndex],
			)
		}
		if beaconState.Balances[transfer.RecipientIndex] < params.BeaconConfig().MinDepositAmount &&
			0 < beaconState.Balances[transfer.RecipientIndex] {
			return nil, fmt.Errorf(
				"recipient balance below critical level: %v",
				beaconState.Balances[transfer.RecipientIndex],
			)
		}
	}
	return beaconState, nil
}

func verifyTransfer(beaconState *pb.BeaconState, transfer *ethpb.Transfer) error {
	if transfer.SenderIndex > uint64(len(beaconState.Validators)) {
		return errors.New("transfer sender index out of bounds in validator registry")
	}

	maxVal := transfer.Fee
	if transfer.Amount > maxVal {
		maxVal = transfer.Amount
	}
	if transfer.Amount+transfer.Fee > maxVal {
		maxVal = transfer.Amount + transfer.Fee
	}
	sender := beaconState.Validators[transfer.SenderIndex]
	senderBalance := beaconState.Balances[transfer.SenderIndex]
	// Verify the balance the covers amount and fee (with overflow protection).
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
	hashed := hashutil.Hash(transfer.SenderWithdrawalPublicKey)
	buf = append(buf, hashed[:][1:]...)
	if !bytes.Equal(sender.WithdrawalCredentials, buf) {
		return fmt.Errorf("invalid public key, expected %v, received %v", buf, sender.WithdrawalCredentials)
	}

	domain := helpers.Domain(beaconState, helpers.CurrentEpoch(beaconState), params.BeaconConfig().DomainTransfer)
	if err := verifySigningRoot(transfer, transfer.SenderWithdrawalPublicKey, transfer.Signature, domain); err != nil {
		return errors.Wrap(err, "could not verify transfer signature")
	}
	return nil
}

// ClearEth1DataVoteCache clears the eth1 data vote count cache.
func ClearEth1DataVoteCache() {
	eth1DataCache = cache.NewEth1DataVoteCache()
}
