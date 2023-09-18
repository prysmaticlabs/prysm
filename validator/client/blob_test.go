package client

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/signing"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func Test_validator_signBlob(t *testing.T) {
	v, m, vk, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().
		DomainData(gomock.Any(), // ctx
			&ethpb.DomainRequest{
				Domain: params.BeaconConfig().DomainBlobSidecar[:],
			}). // epoch
		Return(&ethpb.DomainResponse{
			SignatureDomain: bytesutil.PadTo([]byte("signatureDomain"), 32),
		}, nil)

	blob := &ethpb.BlobSidecar{
		BlockRoot:       bytesutil.PadTo([]byte("blockRoot"), 32),
		Index:           1,
		Slot:            2,
		BlockParentRoot: bytesutil.PadTo([]byte("blockParentRoot"), 32),
		ProposerIndex:   3,
		Blob:            bytesutil.PadTo([]byte("blob"), fieldparams.BlobLength),
		KzgCommitment:   bytesutil.PadTo([]byte("kzgCommitment"), 48),
		KzgProof:        bytesutil.PadTo([]byte("kzgPRoof"), 48),
	}
	ctx := context.Background()
	sig, err := v.signBlob(ctx, blob, [48]byte(vk.PublicKey().Marshal()))
	require.NoError(t, err)
	pb, err := bls.PublicKeyFromBytes(vk.PublicKey().Marshal())
	require.NoError(t, err)
	signature, err := bls.SignatureFromBytes(sig)
	require.NoError(t, err)
	sr, err := signing.ComputeSigningRoot(blob, bytesutil.PadTo([]byte("signatureDomain"), 32))
	require.NoError(t, err)

	require.Equal(t, true, signature.Verify(pb, sr[:]))
}

func TestValidatorSignBlindBlob(t *testing.T) {
	// Setup
	v, m, vk, finish := setup(t)
	defer finish()

	const domainSignature = "signatureDomain"

	// Mock expectations
	m.validatorClient.EXPECT().
		DomainData(gomock.Any(), // context
			&ethpb.DomainRequest{
				Domain: params.BeaconConfig().DomainBlobSidecar[:],
			}).
		Return(&ethpb.DomainResponse{
			SignatureDomain: bytesutil.PadTo([]byte(domainSignature), 32),
		}, nil)

	blobData := &ethpb.BlindedBlobSidecar{
		BlockRoot:       bytesutil.PadTo([]byte("blockRoot"), 32),
		Index:           1,
		Slot:            2,
		BlockParentRoot: bytesutil.PadTo([]byte("blockParentRoot"), 32),
		ProposerIndex:   3,
		BlobRoot:        bytesutil.PadTo([]byte("blobRoot"), 32),
		KzgCommitment:   bytesutil.PadTo([]byte("kzgCommitment"), 48),
		KzgProof:        bytesutil.PadTo([]byte("kzgProof"), 48),
	}

	ctx := context.Background()

	// Test signature creation and validation
	signatureBytes, err := v.signBlindBlob(ctx, blobData, [48]byte(vk.PublicKey().Marshal()))
	require.NoError(t, err)

	publicKey, err := bls.PublicKeyFromBytes(vk.PublicKey().Marshal())
	require.NoError(t, err)

	signature, err := bls.SignatureFromBytes(signatureBytes)
	require.NoError(t, err)

	signingRoot, err := signing.ComputeSigningRoot(blobData, bytesutil.PadTo([]byte(domainSignature), 32))
	require.NoError(t, err)

	// Assert that the signature is valid
	require.Equal(t, true, signature.Verify(publicKey, signingRoot[:]))
}
