package blocks

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"sort"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var log = logrus.WithField("prefix", "blocks")

var eth1DataCache = cache.NewEth1DataVoteCache()

// Deprecated: This method uses deprecated ssz.SigningRoot.
func verifyDepositDataSigningRoot(obj *ethpb.Deposit_Data, pub []byte, signature []byte, domain []byte) error {
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
	sigRoot := &pb.SigningRoot{
		ObjectRoot: root[:],
		Domain:     domain,
	}
	ctrRoot, err := ssz.HashTreeRoot(sigRoot)
	if err != nil {
		return errors.Wrap(err, "could not get container root")
	}
	if !sig.Verify(ctrRoot[:], publicKey) {
		return helpers.ErrSigFailedToVerify
	}
	return nil
}

func verifySignature(signedData []byte, pub []byte, signature []byte, domain []byte) error {
	publicKey, err := bls.PublicKeyFromBytes(pub)
	if err != nil {
		return errors.Wrap(err, "could not convert bytes to public key")
	}
	sig, err := bls.SignatureFromBytes(signature)
	if err != nil {
		return errors.Wrap(err, "could not convert bytes to signature")
	}
	ctr := &pb.SigningRoot{
		ObjectRoot: signedData,
		Domain:     domain,
	}
	root, err := ssz.HashTreeRoot(ctr)
	if err != nil {
		return errors.Wrap(err, "could not hash container")
	}
	if !sig.Verify(root[:], publicKey) {
		return helpers.ErrSigFailedToVerify
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
//    if state.eth1_data_votes.count(body.eth1_data) * 2 > EPOCHS_PER_ETH1_VOTING_PERIOD * SLOTS_PER_EPOCH:
//        state.latest_eth1_data = body.eth1_data
func ProcessEth1DataInBlock(beaconState *stateTrie.BeaconState, block *ethpb.BeaconBlock) (*stateTrie.BeaconState, error) {
	if beaconState == nil {
		return nil, errors.New("nil state")
	}
	if block == nil || block.Body == nil {
		return nil, errors.New("nil block or block withought body")
	}
	if err := beaconState.AppendEth1DataVotes(block.Body.Eth1Data); err != nil {
		return nil, err
	}
	hasSupport, err := Eth1DataHasEnoughSupport(beaconState, block.Body.Eth1Data)
	if err != nil {
		return nil, err
	}
	if hasSupport {
		if err := beaconState.SetEth1Data(block.Body.Eth1Data); err != nil {
			return nil, err
		}
	}
	return beaconState, nil
}

func areEth1DataEqual(a, b *ethpb.Eth1Data) bool {
	if a == nil || b == nil {
		return false
	}
	return a.DepositCount == b.DepositCount &&
		bytes.Equal(a.BlockHash, b.BlockHash) &&
		bytes.Equal(a.DepositRoot, b.DepositRoot)
}

// Eth1DataHasEnoughSupport returns true when the given eth1data has more than 50% votes in the
// eth1 voting period. A vote is cast by including eth1data in a block and part of state processing
// appends eth1data to the state in the Eth1DataVotes list. Iterating through this list checks the
// votes to see if they match the eth1data.
func Eth1DataHasEnoughSupport(beaconState *stateTrie.BeaconState, data *ethpb.Eth1Data) (bool, error) {
	voteCount := uint64(0)
	var eth1DataHash [32]byte
	var err error
	data = stateTrie.CopyETH1Data(data)
	if featureconfig.Get().EnableEth1DataVoteCache {
		eth1DataHash, err = hashutil.HashProto(data)
		if err != nil {
			return false, errors.Wrap(err, "could not hash eth1data")
		}
		voteCount, err = eth1DataCache.Eth1DataVote(eth1DataHash)
		if err != nil {
			return false, errors.Wrap(err, "could not retrieve eth1 data vote cache")
		}
	}
	if voteCount == 0 {
		for _, vote := range beaconState.Eth1DataVotes() {
			if areEth1DataEqual(vote, data) {
				voteCount++
			}
		}
	} else {
		voteCount++
	}

	if featureconfig.Get().EnableEth1DataVoteCache {
		if err := eth1DataCache.AddEth1DataVote(&cache.Eth1DataVote{
			Eth1DataHash: eth1DataHash,
			VoteCount:    voteCount,
		}); err != nil {
			return false, errors.Wrap(err, "could not save eth1 data vote cache")
		}
	}

	// If 50+% majority converged on the same eth1data, then it has enough support to update the
	// state.
	support := params.BeaconConfig().EpochsPerEth1VotingPeriod * params.BeaconConfig().SlotsPerEpoch
	return voteCount*2 > support, nil
}

// ProcessBlockHeader validates a block by its header.
//
// Spec pseudocode definition:
//
//  def process_block_header(state: BeaconState, block: BeaconBlock) -> None:
//    # Verify that the slots match
//    assert block.slot == state.slot
//     # Verify that proposer index is the correct index
//    assert block.proposer_index == get_beacon_proposer_index(state)
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
	beaconState *stateTrie.BeaconState,
	block *ethpb.SignedBeaconBlock,
) (*stateTrie.BeaconState, error) {
	beaconState, err := ProcessBlockHeaderNoVerify(beaconState, block.Block)
	if err != nil {
		return nil, err
	}

	// Verify proposer signature.
	if err := VerifyBlockHeaderSignature(beaconState, block); err != nil {
		return nil, err
	}

	return beaconState, nil
}

// VerifyBlockHeaderSignature verifies the proposer signature of a beacon block.
func VerifyBlockHeaderSignature(beaconState *stateTrie.BeaconState, block *ethpb.SignedBeaconBlock) error {
	proposer, err := beaconState.ValidatorAtIndex(block.Block.ProposerIndex)
	if err != nil {
		return err
	}

	currentEpoch := helpers.SlotToEpoch(beaconState.Slot())
	domain, err := helpers.Domain(beaconState.Fork(), currentEpoch, params.BeaconConfig().DomainBeaconProposer, beaconState.GenesisValidatorRoot())
	if err != nil {
		return err
	}
	return helpers.VerifyBlockSigningRoot(block.Block, proposer.PublicKey, block.Signature, domain)
}

// ProcessBlockHeaderNoVerify validates a block by its header but skips proposer
// signature verification.
//
// WARNING: This method does not verify proposer signature. This is used for proposer to compute state root
// using a unsigned block.
//
// Spec pseudocode definition:
//  def process_block_header(state: BeaconState, block: BeaconBlock) -> None:
//    # Verify that the slots match
//    assert block.slot == state.slot
//     # Verify that proposer index is the correct index
//    assert block.proposer_index == get_beacon_proposer_index(state)
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
	beaconState *stateTrie.BeaconState,
	block *ethpb.BeaconBlock,
) (*stateTrie.BeaconState, error) {
	if block == nil {
		return nil, errors.New("nil block")
	}
	if beaconState.Slot() != block.Slot {
		return nil, fmt.Errorf("state slot: %d is different than block slot: %d", beaconState.Slot(), block.Slot)
	}
	idx, err := helpers.BeaconProposerIndex(beaconState)
	if err != nil {
		return nil, err
	}
	if block.ProposerIndex != idx {
		return nil, fmt.Errorf("proposer index: %d is different than calculated: %d", block.ProposerIndex, idx)
	}
	parentRoot, err := stateutil.BlockHeaderRoot(beaconState.LatestBlockHeader())
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(block.ParentRoot, parentRoot[:]) {
		return nil, fmt.Errorf(
			"parent root %#x does not match the latest block header signing root in state %#x",
			block.ParentRoot, parentRoot)
	}

	proposer, err := beaconState.ValidatorAtIndex(idx)
	if err != nil {
		return nil, err
	}
	if proposer.Slashed {
		return nil, fmt.Errorf("proposer at index %d was previously slashed", idx)
	}

	bodyRoot, err := stateutil.BlockBodyRoot(block.Body)
	if err != nil {
		return nil, err
	}
	if err := beaconState.SetLatestBlockHeader(&ethpb.BeaconBlockHeader{
		Slot:          block.Slot,
		ProposerIndex: block.ProposerIndex,
		ParentRoot:    block.ParentRoot,
		StateRoot:     params.BeaconConfig().ZeroHash[:],
		BodyRoot:      bodyRoot[:],
	}); err != nil {
		return nil, err
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
	beaconState *stateTrie.BeaconState,
	body *ethpb.BeaconBlockBody,
) (*stateTrie.BeaconState, error) {
	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	if err != nil {
		return nil, errors.Wrap(err, "could not get beacon proposer index")
	}
	proposerPub := beaconState.PubkeyAtIndex(proposerIdx)

	currentEpoch := helpers.SlotToEpoch(beaconState.Slot())
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, currentEpoch)

	domain, err := helpers.Domain(beaconState.Fork(), currentEpoch, params.BeaconConfig().DomainRandao, beaconState.GenesisValidatorRoot())
	if err != nil {
		return nil, err
	}
	if err := verifySignature(buf, proposerPub[:], body.RandaoReveal, domain); err != nil {
		return nil, errors.Wrap(err, "could not verify block randao")
	}

	beaconState, err = ProcessRandaoNoVerify(beaconState, body)
	if err != nil {
		return nil, errors.Wrap(err, "could not process randao")
	}
	return beaconState, nil
}

// ProcessRandaoNoVerify generates a new randao mix to update
// in the beacon state's latest randao mixes slice.
//
// Spec pseudocode definition:
//     # Mix it in
//     state.latest_randao_mixes[get_current_epoch(state) % LATEST_RANDAO_MIXES_LENGTH] = (
//         xor(get_randao_mix(state, get_current_epoch(state)),
//             hash(body.randao_reveal))
//     )
func ProcessRandaoNoVerify(
	beaconState *stateTrie.BeaconState,
	body *ethpb.BeaconBlockBody,
) (*stateTrie.BeaconState, error) {
	currentEpoch := helpers.SlotToEpoch(beaconState.Slot())
	// If block randao passed verification, we XOR the state's latest randao mix with the block's
	// randao and update the state's corresponding latest randao mix value.
	latestMixesLength := params.BeaconConfig().EpochsPerHistoricalVector
	latestMixSlice, err := beaconState.RandaoMixAtIndex(currentEpoch % latestMixesLength)
	if err != nil {
		return nil, err
	}
	blockRandaoReveal := hashutil.Hash(body.RandaoReveal)
	if len(blockRandaoReveal) != len(latestMixSlice) {
		return nil, errors.New("blockRandaoReveal length doesnt match latestMixSlice length")
	}
	for i, x := range blockRandaoReveal {
		latestMixSlice[i] ^= x
	}
	if err := beaconState.UpdateRandaoMixesAtIndex(currentEpoch%latestMixesLength, latestMixSlice); err != nil {
		return nil, err
	}
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
//    # Verify slots match
//    assert proposer_slashing.header_1.slot == proposer_slashing.header_2.slot
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
	ctx context.Context,
	beaconState *stateTrie.BeaconState,
	body *ethpb.BeaconBlockBody,
) (*stateTrie.BeaconState, error) {
	var err error
	for idx, slashing := range body.ProposerSlashings {
		if slashing == nil {
			return nil, errors.New("nil proposer slashings in block body")
		}
		if err = VerifyProposerSlashing(beaconState, slashing); err != nil {
			return nil, errors.Wrapf(err, "could not verify proposer slashing %d", idx)
		}
		beaconState, err = v.SlashValidator(
			beaconState, slashing.Header_1.Header.ProposerIndex, 0, /* proposer is whistleblower */
		)
		if err != nil {
			return nil, errors.Wrapf(err, "could not slash proposer index %d", slashing.Header_1.Header.ProposerIndex)
		}
	}
	return beaconState, nil
}

// VerifyProposerSlashing verifies that the data provided from slashing is valid.
func VerifyProposerSlashing(
	beaconState *stateTrie.BeaconState,
	slashing *ethpb.ProposerSlashing,
) error {
	if slashing.Header_1 == nil || slashing.Header_1.Header == nil || slashing.Header_2 == nil || slashing.Header_2.Header == nil {
		return errors.New("nil header cannot be verified")
	}
	if slashing.Header_1.Header.Slot != slashing.Header_2.Header.Slot {
		return fmt.Errorf("mismatched header slots, received %d == %d", slashing.Header_1.Header.Slot, slashing.Header_2.Header.Slot)
	}
	if slashing.Header_1.Header.ProposerIndex != slashing.Header_2.Header.ProposerIndex {
		return fmt.Errorf("mismatched indices, received %d == %d", slashing.Header_1.Header.ProposerIndex, slashing.Header_2.Header.ProposerIndex)
	}
	if proto.Equal(slashing.Header_1, slashing.Header_2) {
		return errors.New("expected slashing headers to differ")
	}
	proposer, err := beaconState.ValidatorAtIndex(slashing.Header_1.Header.ProposerIndex)
	if err != nil {
		return err
	}
	if !helpers.IsSlashableValidator(proposer, helpers.SlotToEpoch(beaconState.Slot())) {
		return fmt.Errorf("validator with key %#x is not slashable", proposer.PublicKey)
	}
	// Using headerEpoch1 here because both of the headers should have the same epoch.
	domain, err := helpers.Domain(beaconState.Fork(), helpers.SlotToEpoch(slashing.Header_1.Header.Slot), params.BeaconConfig().DomainBeaconProposer, beaconState.GenesisValidatorRoot())
	if err != nil {
		return err
	}
	headers := []*ethpb.SignedBeaconBlockHeader{slashing.Header_1, slashing.Header_2}
	for _, header := range headers {
		if err := helpers.VerifySigningRoot(header.Header, proposer.PublicKey, header.Signature, domain); err != nil {
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
//    attestation_1 = attester_slashing.attestation_1
//    attestation_2 = attester_slashing.attestation_2
//    assert is_slashable_attestation_data(attestation_1.data, attestation_2.data)
//    assert is_valid_indexed_attestation(state, attestation_1)
//    assert is_valid_indexed_attestation(state, attestation_2)
//
//    slashed_any = False
//    indices = set(attestation_1.attesting_indices).intersection(attestation_2.attesting_indices)
//    for index in sorted(indices):
//        if is_slashable_validator(state.validators[index], get_current_epoch(state)):
//            slash_validator(state, index)
//            slashed_any = True
//    assert slashed_any
func ProcessAttesterSlashings(
	ctx context.Context,
	beaconState *stateTrie.BeaconState,
	body *ethpb.BeaconBlockBody,
) (*stateTrie.BeaconState, error) {
	for idx, slashing := range body.AttesterSlashings {
		if err := VerifyAttesterSlashing(ctx, beaconState, slashing); err != nil {
			return nil, errors.Wrapf(err, "could not verify attester slashing %d", idx)
		}
		slashableIndices := slashableAttesterIndices(slashing)
		sort.SliceStable(slashableIndices, func(i, j int) bool {
			return slashableIndices[i] < slashableIndices[j]
		})
		currentEpoch := helpers.SlotToEpoch(beaconState.Slot())
		var err error
		var slashedAny bool
		var val *ethpb.Validator
		for _, validatorIndex := range slashableIndices {
			val, err = beaconState.ValidatorAtIndex(validatorIndex)
			if err != nil {
				return nil, err
			}
			if helpers.IsSlashableValidator(val, currentEpoch) {
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
func VerifyAttesterSlashing(ctx context.Context, beaconState *stateTrie.BeaconState, slashing *ethpb.AttesterSlashing) error {
	if slashing == nil {
		return errors.New("nil slashing")
	}
	if slashing.Attestation_1 == nil || slashing.Attestation_2 == nil {
		return errors.New("nil attestation")
	}
	if slashing.Attestation_1.Data == nil || slashing.Attestation_2.Data == nil {
		return errors.New("nil attestation data")
	}
	att1 := slashing.Attestation_1
	att2 := slashing.Attestation_2
	data1 := att1.Data
	data2 := att2.Data
	if !IsSlashableAttestationData(data1, data2) {
		return errors.New("attestations are not slashable")
	}
	if err := VerifyIndexedAttestation(ctx, beaconState, att1); err != nil {
		return errors.Wrap(err, "could not validate indexed attestation")
	}
	if err := VerifyIndexedAttestation(ctx, beaconState, att2); err != nil {
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
	if data1 == nil || data2 == nil || data1.Target == nil || data2.Target == nil || data1.Source == nil || data2.Source == nil {
		return false
	}
	isDoubleVote := !proto.Equal(data1, data2) && data1.Target.Epoch == data2.Target.Epoch
	isSurroundVote := data1.Source.Epoch < data2.Source.Epoch && data2.Target.Epoch < data1.Target.Epoch
	return isDoubleVote || isSurroundVote
}

func slashableAttesterIndices(slashing *ethpb.AttesterSlashing) []uint64 {
	if slashing == nil || slashing.Attestation_1 == nil || slashing.Attestation_2 == nil {
		return nil
	}
	indices1 := slashing.Attestation_1.AttestingIndices
	indices2 := slashing.Attestation_2.AttestingIndices
	return sliceutil.IntersectionUint64(indices1, indices2)
}

// ProcessAttestations applies processing operations to a block's inner attestation
// records. This function returns a list of pending attestations which can then be
// appended to the BeaconState's latest attestations.
func ProcessAttestations(
	ctx context.Context,
	beaconState *stateTrie.BeaconState,
	body *ethpb.BeaconBlockBody,
) (*stateTrie.BeaconState, error) {
	var err error
	for idx, attestation := range body.Attestations {
		beaconState, err = ProcessAttestation(ctx, beaconState, attestation)
		if err != nil {
			return nil, errors.Wrapf(err, "could not verify attestation at index %d in block", idx)
		}
	}
	return beaconState, nil
}

// ProcessAttestationsNoVerify applies processing operations to a block's inner attestation
// records. The only difference would be that the attestation signature would not be verified.
func ProcessAttestationsNoVerify(
	ctx context.Context,
	beaconState *stateTrie.BeaconState,
	body *ethpb.BeaconBlockBody,
) (*stateTrie.BeaconState, error) {
	var err error
	for idx, attestation := range body.Attestations {
		beaconState, err = ProcessAttestationNoVerify(ctx, beaconState, attestation)
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
//    data = attestation.data
//    assert data.index < get_committee_count_at_slot(state, data.slot)
//    assert data.target.epoch in (get_previous_epoch(state), get_current_epoch(state))
//    assert data.target.epoch == compute_epoch_at_slot(data.slot)
//    assert data.slot + MIN_ATTESTATION_INCLUSION_DELAY <= state.slot <= data.slot + SLOTS_PER_EPOCH
//
//    committee = get_beacon_committee(state, data.slot, data.index)
//    assert len(attestation.aggregation_bits) == len(committee)
//
//    pending_attestation = PendingAttestation(
//        data=data,
//        aggregation_bits=attestation.aggregation_bits,
//        inclusion_delay=state.slot - data.slot,
//        proposer_index=get_beacon_proposer_index(state),
//    )
//
//    if data.target.epoch == get_current_epoch(state):
//        assert data.source == state.current_justified_checkpoint
//        state.current_epoch_attestations.append(pending_attestation)
//    else:
//        assert data.source == state.previous_justified_checkpoint
//        state.previous_epoch_attestations.append(pending_attestation)
//
//    # Check signature
//    assert is_valid_indexed_attestation(state, get_indexed_attestation(state, attestation))
func ProcessAttestation(
	ctx context.Context,
	beaconState *stateTrie.BeaconState,
	att *ethpb.Attestation,
) (*stateTrie.BeaconState, error) {
	beaconState, err := ProcessAttestationNoVerify(ctx, beaconState, att)
	if err != nil {
		return nil, err
	}
	return beaconState, VerifyAttestation(ctx, beaconState, att)
}

// ProcessAttestationNoVerify processes the attestation without verifying the attestation signature. This
// method is used to validate attestations whose signatures have already been verified.
func ProcessAttestationNoVerify(
	ctx context.Context,
	beaconState *stateTrie.BeaconState,
	att *ethpb.Attestation,
) (*stateTrie.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "core.ProcessAttestationNoVerify")
	defer span.End()

	if att == nil || att.Data == nil || att.Data.Target == nil {
		return nil, errors.New("nil attestation data target")
	}

	currEpoch := helpers.SlotToEpoch(beaconState.Slot())
	var prevEpoch uint64
	if currEpoch == 0 {
		prevEpoch = 0
	} else {
		prevEpoch = currEpoch - 1
	}
	data := att.Data
	if data.Target.Epoch != prevEpoch && data.Target.Epoch != currEpoch {
		return nil, fmt.Errorf(
			"expected target epoch (%d) to be the previous epoch (%d) or the current epoch (%d)",
			data.Target.Epoch,
			prevEpoch,
			currEpoch,
		)
	}
	if helpers.SlotToEpoch(data.Slot) != data.Target.Epoch {
		return nil, fmt.Errorf("data slot is not in the same epoch as target %d != %d", helpers.SlotToEpoch(data.Slot), data.Target.Epoch)
	}

	s := att.Data.Slot
	minInclusionCheck := s+params.BeaconConfig().MinAttestationInclusionDelay <= beaconState.Slot()
	epochInclusionCheck := beaconState.Slot() <= s+params.BeaconConfig().SlotsPerEpoch
	if !minInclusionCheck {
		return nil, fmt.Errorf(
			"attestation slot %d + inclusion delay %d > state slot %d",
			s,
			params.BeaconConfig().MinAttestationInclusionDelay,
			beaconState.Slot(),
		)
	}
	if !epochInclusionCheck {
		return nil, fmt.Errorf(
			"state slot %d > attestation slot %d + SLOTS_PER_EPOCH %d",
			beaconState.Slot(),
			s,
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
		InclusionDelay:  beaconState.Slot() - s,
		ProposerIndex:   proposerIndex,
	}

	var ffgSourceEpoch uint64
	var ffgSourceRoot []byte
	var ffgTargetEpoch uint64
	if data.Target.Epoch == currEpoch {
		ffgSourceEpoch = beaconState.CurrentJustifiedCheckpoint().Epoch
		ffgSourceRoot = beaconState.CurrentJustifiedCheckpoint().Root
		ffgTargetEpoch = currEpoch
		if err := beaconState.AppendCurrentEpochAttestations(pendingAtt); err != nil {
			return nil, err
		}
	} else {
		ffgSourceEpoch = beaconState.PreviousJustifiedCheckpoint().Epoch
		ffgSourceRoot = beaconState.PreviousJustifiedCheckpoint().Root
		ffgTargetEpoch = prevEpoch
		if err := beaconState.AppendPreviousEpochAttestations(pendingAtt); err != nil {
			return nil, err
		}
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

	return beaconState, nil
}

// VerifyIndexedAttestation determines the validity of an indexed attestation.
//
// Spec pseudocode definition:
//  def is_valid_indexed_attestation(state: BeaconState, indexed_attestation: IndexedAttestation) -> bool:
//    """
//    Check if ``indexed_attestation`` has valid indices and signature.
//    """
//    indices = indexed_attestation.attesting_indices
//
//    # Verify max number of indices
//    if not len(indices) <= MAX_VALIDATORS_PER_COMMITTEE:
//        return False
//    # Verify indices are sorted and unique
//        if not indices == sorted(set(indices)):
//    # Verify aggregate signature
//    if not bls_verify(
//        pubkey=bls_aggregate_pubkeys([state.validators[i].pubkey for i in indices]),
//        message_hash=hash_tree_root(indexed_attestation.data),
//        signature=indexed_attestation.signature,
//        domain=get_domain(state, DOMAIN_BEACON_ATTESTER, indexed_attestation.data.target.epoch),
//    ):
//        return False
//    return True
func VerifyIndexedAttestation(ctx context.Context, beaconState *stateTrie.BeaconState, indexedAtt *ethpb.IndexedAttestation) error {
	ctx, span := trace.StartSpan(ctx, "core.VerifyIndexedAttestation")
	defer span.End()

	if err := attestationutil.IsValidAttestationIndices(ctx, indexedAtt); err != nil {
		return err
	}
	domain, err := helpers.Domain(beaconState.Fork(), indexedAtt.Data.Target.Epoch, params.BeaconConfig().DomainBeaconAttester, beaconState.GenesisValidatorRoot())
	if err != nil {
		return err
	}
	indices := indexedAtt.AttestingIndices
	pubkeys := []*bls.PublicKey{}
	if len(indices) > 0 {
		for i := 0; i < len(indices); i++ {
			pubkeyAtIdx := beaconState.PubkeyAtIndex(indices[i])
			pk, err := bls.PublicKeyFromBytes(pubkeyAtIdx[:])
			if err != nil {
				return errors.Wrap(err, "could not deserialize validator public key")
			}
			pubkeys = append(pubkeys, pk)
		}
	}
	return attestationutil.VerifyIndexedAttestationSig(ctx, indexedAtt, pubkeys, domain)

}

// VerifyAttestation converts and attestation into an indexed attestation and verifies
// the signature in that attestation.
func VerifyAttestation(ctx context.Context, beaconState *stateTrie.BeaconState, att *ethpb.Attestation) error {
	if att == nil || att.Data == nil {
		return fmt.Errorf("nil or missing attestation data: %v", att)
	}
	committee, err := helpers.BeaconCommitteeFromState(beaconState, att.Data.Slot, att.Data.CommitteeIndex)
	if err != nil {
		return err
	}
	indexedAtt := attestationutil.ConvertToIndexed(ctx, att, committee)
	return VerifyIndexedAttestation(ctx, beaconState, indexedAtt)
}

// ProcessDeposits is one of the operations performed on each processed
// beacon block to verify queued validators from the Ethereum 1.0 Deposit Contract
// into the beacon chain.
//
// Spec pseudocode definition:
//   For each deposit in block.body.deposits:
//     process_deposit(state, deposit)
func ProcessDeposits(
	ctx context.Context,
	beaconState *stateTrie.BeaconState,
	body *ethpb.BeaconBlockBody,
) (*stateTrie.BeaconState, error) {
	var err error
	deposits := body.Deposits
	for _, deposit := range deposits {
		if deposit == nil || deposit.Data == nil {
			return nil, errors.New("got a nil deposit in block")
		}
		beaconState, err = ProcessDeposit(beaconState, deposit)
		if err != nil {
			return nil, errors.Wrapf(err, "could not process deposit from %#x", bytesutil.Trunc(deposit.Data.PublicKey))
		}
	}
	return beaconState, nil
}

// ProcessPreGenesisDeposit processes a deposit for the beacon state before chainstart.
func ProcessPreGenesisDeposit(
	ctx context.Context,
	beaconState *stateTrie.BeaconState,
	deposit *ethpb.Deposit,
) (*stateTrie.BeaconState, error) {
	var err error
	beaconState, err = ProcessDeposit(beaconState, deposit)
	if err != nil {
		return nil, errors.Wrap(err, "could not process deposit")
	}
	pubkey := deposit.Data.PublicKey
	index, ok := beaconState.ValidatorIndexByPubkey(bytesutil.ToBytes48(pubkey))
	if !ok {
		return beaconState, nil
	}
	balance, err := beaconState.BalanceAtIndex(index)
	if err != nil {
		return nil, err
	}
	validator, err := beaconState.ValidatorAtIndex(index)
	if err != nil {
		return nil, err
	}
	validator.EffectiveBalance = mathutil.Min(balance-balance%params.BeaconConfig().EffectiveBalanceIncrement, params.BeaconConfig().MaxEffectiveBalance)
	if validator.EffectiveBalance ==
		params.BeaconConfig().MaxEffectiveBalance {
		validator.ActivationEligibilityEpoch = 0
		validator.ActivationEpoch = 0
	}
	if err := beaconState.UpdateValidatorAtIndex(uint64(index), validator); err != nil {
		return nil, err
	}
	return beaconState, nil
}

// ProcessDeposit takes in a deposit object and inserts it
// into the registry as a new validator or balance change.
//
// Spec pseudocode definition:
// def process_deposit(state: BeaconState, deposit: Deposit) -> None:
//    # Verify the Merkle branch
//    assert is_valid_merkle_branch(
//        leaf=hash_tree_root(deposit.data),
//        branch=deposit.proof,
//        depth=DEPOSIT_CONTRACT_TREE_DEPTH + 1,  # Add 1 for the `List` length mix-in
//        index=state.eth1_deposit_index,
//        root=state.eth1_data.deposit_root,
//    )
//
//    # Deposits must be processed in order
//    state.eth1_deposit_index += 1
//
//    pubkey = deposit.data.pubkey
//    amount = deposit.data.amount
//    validator_pubkeys = [v.pubkey for v in state.validators]
//    if pubkey not in validator_pubkeys:
//        # Verify the deposit signature (proof of possession) for new validators.
//        # Note: The deposit contract does not check signatures.
//        # Note: Deposits are valid across forks, thus the deposit domain is retrieved directly from `compute_domain`.
//        domain = compute_domain(DOMAIN_DEPOSIT)
//        if not bls_verify(pubkey, signing_root(deposit.data), deposit.data.signature, domain):
//            return
//
//        # Add validator and balance entries
//        state.validators.append(Validator(
//            pubkey=pubkey,
//            withdrawal_credentials=deposit.data.withdrawal_credentials,
//            activation_eligibility_epoch=FAR_FUTURE_EPOCH,
//            activation_epoch=FAR_FUTURE_EPOCH,
//            exit_epoch=FAR_FUTURE_EPOCH,
//            withdrawable_epoch=FAR_FUTURE_EPOCH,
//            effective_balance=min(amount - amount % EFFECTIVE_BALANCE_INCREMENT, MAX_EFFECTIVE_BALANCE),
//        ))
//        state.balances.append(amount)
//    else:
//        # Increase balance by deposit amount
//        index = ValidatorIndex(validator_pubkeys.index(pubkey))
//        increase_balance(state, index, amount)
func ProcessDeposit(
	beaconState *stateTrie.BeaconState,
	deposit *ethpb.Deposit,
) (*stateTrie.BeaconState, error) {
	if err := verifyDeposit(beaconState, deposit); err != nil {
		if deposit == nil || deposit.Data == nil {
			return nil, err
		}
		return nil, errors.Wrapf(err, "could not verify deposit from %#x", bytesutil.Trunc(deposit.Data.PublicKey))
	}
	if err := beaconState.SetEth1DepositIndex(beaconState.Eth1DepositIndex() + 1); err != nil {
		return nil, err
	}
	pubKey := deposit.Data.PublicKey
	amount := deposit.Data.Amount
	index, ok := beaconState.ValidatorIndexByPubkey(bytesutil.ToBytes48(pubKey))
	numVals := beaconState.NumValidators()
	if !ok {
		domain, err := helpers.ComputeDomain(params.BeaconConfig().DomainDeposit, nil, nil)
		if err != nil {
			return nil, err
		}
		depositSig := deposit.Data.Signature
		if err := verifyDepositDataSigningRoot(deposit.Data, pubKey, depositSig, domain); err != nil {
			// Ignore this error as in the spec pseudo code.
			log.Debugf("Skipping deposit: could not verify deposit data signature: %v", err)
			return beaconState, nil
		}

		effectiveBalance := amount - (amount % params.BeaconConfig().EffectiveBalanceIncrement)
		if params.BeaconConfig().MaxEffectiveBalance < effectiveBalance {
			effectiveBalance = params.BeaconConfig().MaxEffectiveBalance
		}
		if err := beaconState.AppendValidator(&ethpb.Validator{
			PublicKey:                  pubKey,
			WithdrawalCredentials:      deposit.Data.WithdrawalCredentials,
			ActivationEligibilityEpoch: params.BeaconConfig().FarFutureEpoch,
			ActivationEpoch:            params.BeaconConfig().FarFutureEpoch,
			ExitEpoch:                  params.BeaconConfig().FarFutureEpoch,
			WithdrawableEpoch:          params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance:           effectiveBalance,
		}); err != nil {
			return nil, err
		}
		if err := beaconState.AppendBalance(amount); err != nil {
			return nil, err
		}
		numVals++
		beaconState.SetValidatorIndexByPubkey(bytesutil.ToBytes48(pubKey), uint64(numVals-1))
	} else {
		if err := helpers.IncreaseBalance(beaconState, uint64(index), amount); err != nil {
			return nil, err
		}
	}

	return beaconState, nil
}

func verifyDeposit(beaconState *stateTrie.BeaconState, deposit *ethpb.Deposit) error {
	// Verify Merkle proof of deposit and deposit trie root.
	if deposit == nil || deposit.Data == nil {
		return errors.New("received nil deposit or nil deposit data")
	}
	eth1Data := beaconState.Eth1Data()
	if eth1Data == nil {
		return errors.New("received nil eth1data in the beacon state")
	}

	receiptRoot := eth1Data.DepositRoot
	leaf, err := ssz.HashTreeRoot(deposit.Data)
	if err != nil {
		return errors.Wrap(err, "could not tree hash deposit data")
	}
	if ok := trieutil.VerifyMerkleBranch(
		receiptRoot,
		leaf[:],
		int(beaconState.Eth1DepositIndex()),
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
	ctx context.Context,
	beaconState *stateTrie.BeaconState,
	body *ethpb.BeaconBlockBody,
) (*stateTrie.BeaconState, error) {
	exits := body.VoluntaryExits
	for idx, exit := range exits {
		if exit == nil || exit.Exit == nil {
			return nil, errors.New("nil voluntary exit in block body")
		}
		if int(exit.Exit.ValidatorIndex) >= beaconState.NumValidators() {
			return nil, fmt.Errorf(
				"validator index out of bound %d > %d",
				exit.Exit.ValidatorIndex,
				beaconState.NumValidators(),
			)
		}
		val, err := beaconState.ValidatorAtIndex(exit.Exit.ValidatorIndex)
		if err != nil {
			return nil, err
		}
		if err := VerifyExit(val, beaconState.Slot(), beaconState.Fork(), exit, beaconState.GenesisValidatorRoot()); err != nil {
			return nil, errors.Wrapf(err, "could not verify exit %d", idx)
		}
		beaconState, err = v.InitiateValidatorExit(beaconState, exit.Exit.ValidatorIndex)
		if err != nil {
			return nil, err
		}
	}
	return beaconState, nil
}

// ProcessVoluntaryExitsNoVerify processes all the voluntary exits in
// a block body, without verifying their BLS signatures.
func ProcessVoluntaryExitsNoVerify(
	beaconState *stateTrie.BeaconState,
	body *ethpb.BeaconBlockBody,
) (*stateTrie.BeaconState, error) {
	var err error
	exits := body.VoluntaryExits

	for idx, exit := range exits {
		if exit == nil || exit.Exit == nil {
			return nil, errors.New("nil exit")
		}
		beaconState, err = v.InitiateValidatorExit(beaconState, exit.Exit.ValidatorIndex)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to process voluntary exit at index %d", idx)
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
func VerifyExit(validator *ethpb.Validator, currentSlot uint64, fork *pb.Fork, signed *ethpb.SignedVoluntaryExit, genesisRoot []byte) error {
	if signed == nil || signed.Exit == nil {
		return errors.New("nil exit")
	}

	exit := signed.Exit
	currentEpoch := helpers.SlotToEpoch(currentSlot)
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
	domain, err := helpers.Domain(fork, exit.Epoch, params.BeaconConfig().DomainVoluntaryExit, genesisRoot)
	if err != nil {
		return err
	}
	if err := helpers.VerifySigningRoot(exit, validator.PublicKey, signed.Signature, domain); err != nil {
		return helpers.ErrSigFailedToVerify
	}
	return nil
}

// ClearEth1DataVoteCache clears the eth1 data vote count cache.
func ClearEth1DataVoteCache() {
	eth1DataCache = cache.NewEth1DataVoteCache()
}
