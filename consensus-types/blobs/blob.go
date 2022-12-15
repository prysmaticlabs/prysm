package blobs

import (
	"fmt"

	"github.com/protolambda/go-kzg/eth"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	v1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	log "github.com/sirupsen/logrus"
)

type commitmentSequenceImpl [][]byte

func (s commitmentSequenceImpl) At(i int) eth.KZGCommitment {
	var out eth.KZGCommitment
	copy(out[:], s[i])
	return out
}

func (s commitmentSequenceImpl) Len() int {
	return len(s)
}

type BlobImpl []byte

func (b BlobImpl) At(i int) [32]byte {
	var out [32]byte
	copy(out[:], b[i*32:(i+1)*32-1])
	return out
}

func (b BlobImpl) Len() int {
	return len(b) / 32
}

type BlobsSequenceImpl []*v1.Blob

func (s BlobsSequenceImpl) At(i int) eth.Blob {
	return BlobImpl(s[i].Data)
}

func (s BlobsSequenceImpl) Len() int {
	return len(s)
}

// ValidateBlobsSidecar verifies the integrity of a sidecar, returning nil if the blob is valid.
func ValidateBlobsSidecar(slot types.Slot, root [32]byte, commitments [][]byte, sidecar *ethpb.BlobsSidecar) error {
	kzgSidecar := eth.BlobsSidecar{
		BeaconBlockRoot:    eth.Root(bytesutil.ToBytes32(sidecar.BeaconBlockRoot)),
		BeaconBlockSlot:    eth.Slot(sidecar.BeaconBlockSlot),
		Blobs:              BlobsSequenceImpl(sidecar.Blobs),
		KZGAggregatedProof: eth.KZGProof(bytesutil.ToBytes48(sidecar.AggregatedProof)),
	}
	log.WithFields(log.Fields{
		"slot":            slot,
		"root":            fmt.Sprintf("%#x", root),
		"commitments":     len(commitments),
		"sidecarSlot":     sidecar.BeaconBlockSlot,
		"sidecarRoot":     fmt.Sprintf("%#x", sidecar.BeaconBlockRoot),
		"sidecarBlobs":    len(sidecar.Blobs),
		"aggregatedProof": fmt.Sprintf("%#x", sidecar.AggregatedProof),
	}).Infof("Validating blobs sidecar for slot %d", slot)
	return eth.ValidateBlobsSidecar(eth.Slot(slot), root, commitmentSequenceImpl(commitments), kzgSidecar)
}
