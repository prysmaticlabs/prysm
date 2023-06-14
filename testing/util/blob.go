package util

import (
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

// HydrateSignedBlindedBlobSidecar hydrates a signed blinded blob sidecar with correct field length sizes
// to comply with SSZ marshalling and unmarshalling rules.
func HydrateSignedBlindedBlobSidecar(b *ethpb.SignedBlindedBlobSidecar) *ethpb.SignedBlindedBlobSidecar {
	if b.Signature == nil {
		b.Signature = make([]byte, fieldparams.BLSSignatureLength)
	}
	b.Message = HydrateBlindedBlobSidecar(b.Message)
	return b
}

// HydrateBlindedBlobSidecar hydrates a blinded blob sidecar with correct field length sizes
// to comply with SSZ marshalling and unmarshalling rules.
func HydrateBlindedBlobSidecar(b *ethpb.BlindedBlobSidecar) *ethpb.BlindedBlobSidecar {
	if b.BlockRoot == nil {
		b.BlockRoot = make([]byte, fieldparams.RootLength)
	}
	if b.BlockParentRoot == nil {
		b.BlockParentRoot = make([]byte, fieldparams.RootLength)
	}
	if b.KzgCommitment == nil {
		b.KzgCommitment = make([]byte, fieldparams.BLSPubkeyLength)
	}
	if b.KzgProof == nil {
		b.KzgProof = make([]byte, fieldparams.BLSPubkeyLength)
	}
	if b.BlobRoot == nil {
		b.BlobRoot = make([]byte, fieldparams.RootLength)
	}
	return b
}
