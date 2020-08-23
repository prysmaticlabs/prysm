package blocks

import (
	"context"
	"encoding/binary"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
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

// verifies the signature from the raw data, public key and domain provided.
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

// VerifyBlockSignature verifies the proposer signature of a beacon block.
func VerifyBlockSignature(beaconState *stateTrie.BeaconState, block *ethpb.SignedBeaconBlock) error {
	currentEpoch := helpers.SlotToEpoch(beaconState.Slot())
	domain, err := helpers.Domain(beaconState.Fork(), currentEpoch, params.BeaconConfig().DomainBeaconProposer, beaconState.GenesisValidatorRoot())
	if err != nil {
		return err
	}
	proposer, err := beaconState.ValidatorAtIndex(block.Block.ProposerIndex)
	if err != nil {
		return err
	}
	proposerPubKey := proposer.PublicKey
	return helpers.VerifyBlockSigningRoot(block.Block, proposerPubKey[:], block.Signature, domain)
}

// BlockSignatureSet retrieves the block signature set from the provided block and its corresponding state.
func BlockSignatureSet(beaconState *stateTrie.BeaconState, block *ethpb.SignedBeaconBlock) (*bls.SignatureSet, error) {
	currentEpoch := helpers.SlotToEpoch(beaconState.Slot())
	domain, err := helpers.Domain(beaconState.Fork(), currentEpoch, params.BeaconConfig().DomainBeaconProposer, beaconState.GenesisValidatorRoot())
	if err != nil {
		return nil, err
	}
	proposer, err := beaconState.ValidatorAtIndex(block.Block.ProposerIndex)
	if err != nil {
		return nil, err
	}
	proposerPubKey := proposer.PublicKey
	return helpers.RetrieveBlockSignatureSet(block.Block, proposerPubKey, block.Signature, domain)
}

// RandaoSignatureSet retrieves the relevant randao specific signature set object
// from a block and its corresponding state.
func RandaoSignatureSet(beaconState *stateTrie.BeaconState,
	body *ethpb.BeaconBlockBody,
) (*bls.SignatureSet, *stateTrie.BeaconState, error) {
	buf, proposerPub, domain, err := randaoSigningData(beaconState)
	if err != nil {
		return nil, nil, err
	}
	set, err := retrieveSignatureSet(buf, proposerPub[:], body.RandaoReveal, domain)
	if err != nil {
		return nil, nil, err
	}
	return set, beaconState, nil
}

// retrieves the randao related signing data from the state.
func randaoSigningData(beaconState *stateTrie.BeaconState) ([]byte, []byte, []byte, error) {
	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "could not get beacon proposer index")
	}
	proposerPub := beaconState.PubkeyAtIndex(proposerIdx)

	currentEpoch := helpers.SlotToEpoch(beaconState.Slot())
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, currentEpoch)

	domain, err := helpers.Domain(beaconState.Fork(), currentEpoch, params.BeaconConfig().DomainRandao, beaconState.GenesisValidatorRoot())
	if err != nil {
		return nil, nil, nil, err
	}
	return buf, proposerPub[:], domain, nil
}

// Method to break down attestations of the same domain and collect them into a single signature set.
func createAttestationSignatureSet(ctx context.Context, beaconState *stateTrie.BeaconState, atts []*ethpb.Attestation, domain []byte) (*bls.SignatureSet, error) {
	if len(atts) == 0 {
		return nil, nil
	}

	sigs := make([]bls.Signature, len(atts))
	pks := make([]bls.PublicKey, len(atts))
	msgs := make([][32]byte, len(atts))
	for i, a := range atts {
		sig, err := bls.SignatureFromBytes(a.Signature)
		if err != nil {
			return nil, err
		}
		sigs[i] = sig
		c, err := helpers.BeaconCommitteeFromState(beaconState, a.Data.Slot, a.Data.CommitteeIndex)
		if err != nil {
			return nil, err
		}
		ia := attestationutil.ConvertToIndexed(ctx, a, c)
		if err := attestationutil.IsValidAttestationIndices(ctx, ia); err != nil {
			return nil, err
		}
		indices := ia.AttestingIndices
		var pk bls.PublicKey
		for i := 0; i < len(indices); i++ {
			pubkeyAtIdx := beaconState.PubkeyAtIndex(indices[i])
			p, err := bls.PublicKeyFromBytes(pubkeyAtIdx[:])
			if err != nil {
				return nil, errors.Wrap(err, "could not deserialize validator public key")
			}
			if pk == nil {
				pk = p
			} else {
				pk.Aggregate(p)
			}
		}
		pks[i] = pk

		root, err := helpers.ComputeSigningRoot(ia.Data, domain)
		if err != nil {
			return nil, errors.Wrap(err, "could not get signing root of object")
		}
		msgs[i] = root
	}
	return &bls.SignatureSet{
		Signatures: sigs,
		PublicKeys: pks,
		Messages:   msgs,
	}, nil
}

// AttestationSignatureSet retrieves all the related attestation signature data such as the relevant public keys,
// signatures and attestation signing data and collate it into a signature set object.
func AttestationSignatureSet(ctx context.Context, beaconState *stateTrie.BeaconState, atts []*ethpb.Attestation) (*bls.SignatureSet, error) {
	if len(atts) == 0 {
		return bls.NewSet(), nil
	}

	fork := beaconState.Fork()
	gvr := beaconState.GenesisValidatorRoot()
	dt := params.BeaconConfig().DomainBeaconAttester

	// Split attestations by fork. Note: the signature domain will differ based on the fork.
	var preForkAtts []*ethpb.Attestation
	var postForkAtts []*ethpb.Attestation
	for _, a := range atts {
		if helpers.SlotToEpoch(a.Data.Slot) < fork.Epoch {
			preForkAtts = append(preForkAtts, a)
		} else {
			postForkAtts = append(postForkAtts, a)
		}
	}
	set := bls.NewSet()

	// Check attestations from before the fork.
	if fork.Epoch > 0 { // Check to prevent underflow.
		prevDomain, err := helpers.Domain(fork, fork.Epoch-1, dt, gvr)
		if err != nil {
			return nil, err
		}
		aSet, err := createAttestationSignatureSet(ctx, beaconState, preForkAtts, prevDomain)
		if err != nil {
			return nil, err
		}
		set.Join(aSet)
	} else if len(preForkAtts) > 0 {
		// This is a sanity check that preForkAtts were not ignored when fork.Epoch == 0. This
		// condition is not possible, but it doesn't hurt to check anyway.
		return nil, errors.New("some attestations were not verified from previous fork before genesis")
	}

	// Then check attestations from after the fork.
	currDomain, err := helpers.Domain(fork, fork.Epoch, dt, gvr)
	if err != nil {
		return nil, err
	}

	aSet, err := createAttestationSignatureSet(ctx, beaconState, postForkAtts, currDomain)
	if err != nil {
		return nil, err
	}
	return set.Join(aSet), nil
}
