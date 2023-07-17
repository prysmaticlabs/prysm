package util

import (
	"fmt"

	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	v2 "github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

// NewBeaconBlockBellatrix creates a beacon block with minimum marshalable fields.
func NewBeaconBlockBellatrix() *ethpb.SignedBeaconBlockBellatrix {
	return HydrateSignedBeaconBlockBellatrix(&ethpb.SignedBeaconBlockBellatrix{})
}

// NewBlindedBeaconBlockBellatrix creates a blinded beacon block with minimum marshalable fields.
func NewBlindedBeaconBlockBellatrix() *ethpb.SignedBlindedBeaconBlockBellatrix {
	return HydrateSignedBlindedBeaconBlockBellatrix(&ethpb.SignedBlindedBeaconBlockBellatrix{})
}

// NewBlindedBeaconBlockBellatrixV2 creates a blinded beacon block with minimum marshalable fields.
func NewBlindedBeaconBlockBellatrixV2() *v2.SignedBlindedBeaconBlockBellatrix {
	return HydrateV2SignedBlindedBeaconBlockBellatrix(&v2.SignedBlindedBeaconBlockBellatrix{})
}

// NewBeaconBlockCapella creates a beacon block with minimum marshalable fields.
func NewBeaconBlockCapella() *ethpb.SignedBeaconBlockCapella {
	return HydrateSignedBeaconBlockCapella(&ethpb.SignedBeaconBlockCapella{})
}

// NewBlindedBeaconBlockCapella creates a blinded beacon block with minimum marshalable fields.
func NewBlindedBeaconBlockCapella() *ethpb.SignedBlindedBeaconBlockCapella {
	return HydrateSignedBlindedBeaconBlockCapella(&ethpb.SignedBlindedBeaconBlockCapella{})
}

// NewBeaconBlockDeneb creates a beacon block with minimum marshalable fields.
func NewBeaconBlockDeneb() *ethpb.SignedBeaconBlockDeneb {
	return HydrateSignedBeaconBlockDeneb(&ethpb.SignedBeaconBlockDeneb{})
}

// NewBlindedBeaconBlockDeneb creates a blinded beacon block with minimum marshalable fields.
func NewBlindedBeaconBlockDeneb() *ethpb.SignedBlindedBeaconBlockDeneb {
	return HydrateSignedBlindedBeaconBlockDeneb(&ethpb.SignedBlindedBeaconBlockDeneb{})
}

// NewBlindedBlobSidecar creates a signed blinded blob sidecar with minimum marshalable fields.
func NewBlindedBlobSidecar() *ethpb.SignedBlindedBlobSidecar {
	return HydrateSignedBlindedBlobSidecar(&ethpb.SignedBlindedBlobSidecar{})
}

// NewBlindedBeaconBlockCapellaV2 creates a blinded beacon block with minimum marshalable fields.
func NewBlindedBeaconBlockCapellaV2() *v2.SignedBlindedBeaconBlockCapella {
	return HydrateV2SignedBlindedBeaconBlockCapella(&v2.SignedBlindedBeaconBlockCapella{})
}

// NewBeaconBlockContentsDeneb creates a beacon block content including blobs with minimum marshalable fields.
func NewBeaconBlockContentsDeneb(numOfBlobs uint64) (*v2.SignedBeaconBlockContentsDeneb, error) {
	if numOfBlobs > fieldparams.MaxBlobsPerBlock {
		return nil, fmt.Errorf("declared too many blobs: %v", numOfBlobs)
	}
	blobs := make([]*v2.SignedBlobSidecar, numOfBlobs)
	for i := range blobs {
		blobs[i] = &v2.SignedBlobSidecar{
			Message: &v2.BlobSidecar{
				BlockRoot:       make([]byte, fieldparams.RootLength),
				Index:           0,
				Slot:            0,
				BlockParentRoot: make([]byte, fieldparams.RootLength),
				ProposerIndex:   0,
				Blob:            make([]byte, fieldparams.BlobLength),
				KzgCommitment:   make([]byte, fieldparams.BLSPubkeyLength),
				KzgProof:        make([]byte, fieldparams.BLSPubkeyLength),
			},
			Signature: make([]byte, fieldparams.BLSSignatureLength),
		}
	}
	return &v2.SignedBeaconBlockContentsDeneb{
		SignedBlock:        HydrateV2SignedBeaconBlockDeneb(&v2.SignedBeaconBlockDeneb{}),
		SignedBlobSidecars: blobs,
	}, nil
}

// NewBlindedBeaconBlockContentsDeneb creates a blinded beacon block content including blobs with minimum marshalable fields.
func NewBlindedBeaconBlockContentsDeneb(numOfBlobs uint64) (*v2.SignedBlindedBeaconBlockContentsDeneb, error) {
	if numOfBlobs > fieldparams.MaxBlobsPerBlock {
		return nil, fmt.Errorf("declared too many blobs: %v", numOfBlobs)
	}
	blobs := make([]*v2.SignedBlindedBlobSidecar, numOfBlobs)
	for i := range blobs {
		blobs[i] = &v2.SignedBlindedBlobSidecar{
			Message: &v2.BlindedBlobSidecar{
				BlockRoot:       make([]byte, fieldparams.RootLength),
				Index:           0,
				Slot:            0,
				BlockParentRoot: make([]byte, fieldparams.RootLength),
				ProposerIndex:   0,
				BlobRoot:        make([]byte, fieldparams.RootLength),
				KzgCommitment:   make([]byte, fieldparams.BLSPubkeyLength),
				KzgProof:        make([]byte, fieldparams.BLSPubkeyLength),
			},
			Signature: make([]byte, fieldparams.BLSSignatureLength),
		}
	}
	return &v2.SignedBlindedBeaconBlockContentsDeneb{
		SignedBlindedBlock:        HydrateV2SignedBlindedBeaconBlockDeneb(&v2.SignedBlindedBeaconBlockDeneb{}),
		SignedBlindedBlobSidecars: blobs,
	}, nil
}
