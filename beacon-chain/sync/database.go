package sync

import eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"

type BlobSidecarGetter interface {
	BlobSidecar(blockRoot [32]byte, index uint64) (*eth.BlobSidecar, error)
}

type BlobSidecarWriter interface {
	WriteBlobSidecar(blockRoot [32]byte, index uint64, sidecar *eth.BlobSidecar) error
}

type BlobDB interface {
	BlobSidecarGetter
	BlobSidecarWriter
}
