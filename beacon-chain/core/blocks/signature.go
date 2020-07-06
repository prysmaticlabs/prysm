package blocks

import (
	"bytes"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// retrieves the signature set from the raw data, public key,signature and domain provided.
func retrieveSignatureSet(signedData []byte, pub []byte, signature []byte, domain []byte) (*bls.SignatureSet, error) {
	publicKey, err := bls.PublicKeyFromBytes(pub)
	if err != nil {
		return nil, errors.Wrap(err, "could not convert bytes to public key")
	}
	sig, err := bls.SignatureFromBytes(signature)
	if err != nil {
		return nil, errors.Wrap(err, "could not convert bytes to signature")
	}
	signingData := &pb.SigningData{
		ObjectRoot: signedData,
		Domain:     domain,
	}
	root, err := ssz.HashTreeRoot(signingData)
	if err != nil {
		return nil, errors.Wrap(err, "could not hash container")
	}
	return &bls.SignatureSet{
		Signatures: []bls.Signature{sig},
		PublicKeys: []bls.PublicKey{publicKey},
		Messages:   [][32]byte{root},
	}, nil
}

// verifies the signature  from the raw data, public key and domain provided.
func verifySignature(signedData []byte, pub []byte, signature []byte, domain []byte) error {
	set, err := retrieveSignatureSet(signedData, pub, signature, domain)
	if err != nil {
		return err
	}
	if len(set.Signatures) != 1 {
		return errors.Errorf("signature set contains %d signatures instead of 1", len(set.Signatures))
	}
	// We assume only one signature set is returned here.
	sig := set.Signatures[0]
	publicKey := set.PublicKeys[0]
	root := set.Messages[0]
	if !sig.Verify(publicKey, root[:]) {
		return helpers.ErrSigFailedToVerify
	}
	return nil
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
//    assert block.parent_root == hash_tree_root(state.latest_block_header)
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
	if err := VerifyBlockSignature(beaconState, block); err != nil {
		return nil, err
	}

	return beaconState, nil
}

// VerifyBlockSignature verifies the proposer signature of a beacon block.
func VerifyBlockSignature(beaconState *stateTrie.BeaconState, block *ethpb.SignedBeaconBlock) error {
	proposer, err := beaconState.ValidatorAtIndex(block.Block.ProposerIndex)
	if err != nil {
		return err
	}

	currentEpoch := helpers.SlotToEpoch(beaconState.Slot())
	domain, err := helpers.Domain(beaconState.Fork(), currentEpoch, params.BeaconConfig().DomainBeaconProposer, beaconState.GenesisValidatorRoot())
	if err != nil {
		return err
	}
	proposerPubKey := proposer.PublicKey
	return helpers.VerifyBlockSigningRoot(block.Block, proposerPubKey[:], block.Signature, domain)
}

// BlockSignatureSet retrieves the block signature set from the provided block and its corresponding state.
func BlockSignatureSet(beaconState *stateTrie.BeaconState, block *ethpb.SignedBeaconBlock) (*bls.SignatureSet, error) {
	proposer, err := beaconState.ValidatorAtIndex(block.Block.ProposerIndex)
	if err != nil {
		return nil, err
	}

	currentEpoch := helpers.SlotToEpoch(beaconState.Slot())
	domain, err := helpers.Domain(beaconState.Fork(), currentEpoch, params.BeaconConfig().DomainBeaconProposer, beaconState.GenesisValidatorRoot())
	if err != nil {
		return nil, err
	}
	proposerPubKey := proposer.PublicKey
	return helpers.RetrieveBlockSignatureSet(block.Block, proposerPubKey, block.Signature, domain)
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
//    assert block.parent_root == hash_tree_root(state.latest_block_header)
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
	parentHeader := beaconState.LatestBlockHeader()
	if parentHeader.Slot >= block.Slot {
		return nil, fmt.Errorf("block.Slot %d must be greater than state.LatestBlockHeader.Slot %d", block.Slot, parentHeader.Slot)
	}
	parentRoot, err := stateutil.BlockHeaderRoot(parentHeader)
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(block.ParentRoot, parentRoot[:]) {
		return nil, fmt.Errorf(
			"parent root %#x does not match the latest block header signing root in state %#x",
			block.ParentRoot, parentRoot)
	}

	proposer, err := beaconState.ValidatorAtIndexReadOnly(idx)
	if err != nil {
		return nil, err
	}
	if proposer.Slashed() {
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
