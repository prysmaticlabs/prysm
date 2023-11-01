package client

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/signing"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	validatorpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

func (v *validator) signBlob(ctx context.Context, blob *ethpb.DeprecatedBlobSidecar, pubKey [fieldparams.BLSPubkeyLength]byte) ([]byte, error) {
	epoch := slots.ToEpoch(blob.Slot)
	domain, err := v.domainData(ctx, epoch, params.BeaconConfig().DomainBlobSidecar[:])
	if err != nil {
		return nil, errors.Wrap(err, domainDataErr)
	}
	if domain == nil {
		return nil, errors.New(domainDataErr)
	}
	sr, err := signing.ComputeSigningRoot(blob, domain.SignatureDomain)
	if err != nil {
		return nil, errors.Wrap(err, signingRootErr)
	}
	sig, err := v.keyManager.Sign(ctx, &validatorpb.SignRequest{
		PublicKey:       pubKey[:],
		SigningRoot:     sr[:],
		SignatureDomain: domain.SignatureDomain,
		Object:          &validatorpb.SignRequest_Blob{Blob: blob},
		SigningSlot:     blob.Slot,
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not sign block proposal")
	}
	return sig.Marshal(), nil
}

// signBlindBlob signs a given blinded blob sidecar for a specific slot.
// It calculates the signing root for the blob and then uses the key manager to produce the signature.
func (v *validator) signBlindBlob(ctx context.Context, blob *ethpb.BlindedBlobSidecar, pubKey [fieldparams.BLSPubkeyLength]byte) ([]byte, error) {
	epoch := slots.ToEpoch(blob.Slot)

	// Retrieve domain data specific to the epoch and `DOMAIN_BLOB_SIDECAR`.
	domain, err := v.domainData(ctx, epoch, params.BeaconConfig().DomainBlobSidecar[:])
	if err != nil {
		return nil, errors.Wrap(err, domainDataErr)
	}
	if domain == nil {
		return nil, errors.New(domainDataErr)
	}

	// Compute the signing root for the blob.
	sr, err := signing.ComputeSigningRoot(blob, domain.SignatureDomain)
	if err != nil {
		return nil, errors.Wrap(err, signingRootErr)
	}

	// Create a sign request and use the key manager to sign it.
	sig, err := v.keyManager.Sign(ctx, &validatorpb.SignRequest{
		PublicKey:       pubKey[:],
		SigningRoot:     sr[:],
		SignatureDomain: domain.SignatureDomain,
		Object:          &validatorpb.SignRequest_BlindedBlob{BlindedBlob: blob},
		SigningSlot:     blob.Slot,
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not sign blind blob sidecar")
	}
	return sig.Marshal(), nil
}

// signDenebBlobs signs an array of Deneb blobs using the provided public key.
func (v *validator) signDenebBlobs(ctx context.Context, blobs []*ethpb.DeprecatedBlobSidecar, pubKey [48]byte) ([]*ethpb.SignedBlobSidecar, error) {
	signedBlobs := make([]*ethpb.SignedBlobSidecar, len(blobs))
	for i, blob := range blobs {
		blobSig, err := v.signBlob(ctx, blob, pubKey)
		if err != nil {
			return nil, err
		}
		signedBlobs[i] = &ethpb.SignedBlobSidecar{
			Message:   blob,
			Signature: blobSig,
		}
	}
	return signedBlobs, nil
}

// signBlindedDenebBlobs signs an array of blinded Deneb blobs using the provided public key.
func (v *validator) signBlindedDenebBlobs(ctx context.Context, blobs []*ethpb.BlindedBlobSidecar, pubKey [48]byte) ([]*ethpb.SignedBlindedBlobSidecar, error) {
	signedBlindBlobs := make([]*ethpb.SignedBlindedBlobSidecar, len(blobs))
	for i, blob := range blobs {
		blobSig, err := v.signBlindBlob(ctx, blob, pubKey)
		if err != nil {
			return nil, err
		}
		signedBlindBlobs[i] = &ethpb.SignedBlindedBlobSidecar{
			Message:   blob,
			Signature: blobSig,
		}
	}
	return signedBlindBlobs, nil
}
