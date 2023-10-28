package util

import (
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

// HydrateBlobSidecar hydrates a blob sidecar with correct field length sizes
// to comply with SSZ marshalling and unmarshalling rules.
func HydrateBlobSidecar(b *ethpb.BlobSidecar) *ethpb.BlobSidecar {
	if b.SignedBlockHeader == nil {
		b.SignedBlockHeader = HydrateSignedBeaconHeader(&ethpb.SignedBeaconBlockHeader{})
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

// HydrateBlindedBlobSidecar hydrates a blinded blob sidecar with correct field length sizes
// to comply with SSZ marshalling and unmarshalling rules.
func HydrateBlindedBlobSidecar(b *ethpb.BlindedBlobSidecar) *ethpb.BlindedBlobSidecar {
	if b.SignedBlockHeader == nil {
		b.SignedBlockHeader = HydrateSignedBeaconHeader(&ethpb.SignedBeaconBlockHeader{})
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
