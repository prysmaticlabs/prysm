package util

import (
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
)

func NewSignedBlindedBlobSidecarV2() *eth.SignedBlindedBlobSidecar {
	return HydrateSignedBlindedBlobSidecarV2(&eth.SignedBlindedBlobSidecar{})
}

func HydrateSignedBlindedBlobSidecarV2(sidecar *eth.SignedBlindedBlobSidecar) *eth.SignedBlindedBlobSidecar {
	if sidecar == nil {
		sidecar = &eth.SignedBlindedBlobSidecar{}
	}
	if sidecar.Message == nil {
		sidecar.Message = &eth.BlindedBlobSidecar{}
	}
	if sidecar.Message.BlobRoot == nil {
		sidecar.Message.BlobRoot = make([]byte, fieldparams.RootLength)
	}
	if sidecar.Message.BlockParentRoot == nil {
		sidecar.Message.BlockParentRoot = make([]byte, fieldparams.RootLength)
	}
	if sidecar.Message.KzgCommitment == nil {
		sidecar.Message.KzgCommitment = make([]byte, 48)
	}
	if sidecar.Message.KzgProof == nil {
		sidecar.Message.KzgProof = make([]byte, 48)
	}
	if sidecar.Signature == nil {
		sidecar.Signature = make([]byte, fieldparams.BLSSignatureLength)
	}
	return sidecar
}
