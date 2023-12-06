package util

import (
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

// HydrateSignedBlobSidecar hydrates a signed blob sidecar with correct field length sizes
// to comply with SSZ marshalling and unmarshalling rules.
func HydrateSignedBlobSidecar(b *ethpb.SignedBlobSidecar) *ethpb.SignedBlobSidecar {
	if b.Signature == nil {
		b.Signature = make([]byte, fieldparams.BLSSignatureLength)
	}
	if b.Message == nil {
		b.Message = &ethpb.DeprecatedBlobSidecar{}
	}
	b.Message = HydrateBlobSidecar(b.Message)
	return b
}

// HydrateBlobSidecar hydrates a blob sidecar with correct field length sizes
// to comply with SSZ marshalling and unmarshalling rules.
func HydrateBlobSidecar(b *ethpb.DeprecatedBlobSidecar) *ethpb.DeprecatedBlobSidecar {
	if b.BlockRoot == nil {
		b.BlockRoot = make([]byte, fieldparams.RootLength)
	}
	if b.BlockParentRoot == nil {
		b.BlockParentRoot = make([]byte, fieldparams.RootLength)
	}
	if b.Blob == nil {
		b.Blob = make([]byte, fieldparams.BlobLength)
	}
	if b.KzgCommitment == nil {
		b.KzgCommitment = make([]byte, fieldparams.BLSPubkeyLength)
	}
	if b.KzgProof == nil {
		b.KzgProof = make([]byte, fieldparams.BLSPubkeyLength)
	}
	return b
}

// HydrateSignedBlindedBlobSidecar hydrates a signed blinded blob sidecar with correct field length sizes
// to comply with SSZ marshalling and unmarshalling rules.
func HydrateSignedBlindedBlobSidecar(b *ethpb.SignedBlindedBlobSidecar) *ethpb.SignedBlindedBlobSidecar {
	if b.Signature == nil {
		b.Signature = make([]byte, fieldparams.BLSSignatureLength)
	}
	if b.Message == nil {
		b.Message = &ethpb.BlindedBlobSidecar{}
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
