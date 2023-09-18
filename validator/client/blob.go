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

func (v *validator) signBlob(ctx context.Context, blob *ethpb.BlobSidecar, pubKey [fieldparams.BLSPubkeyLength]byte) ([]byte, error) {
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
