package sync

import eth "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"

type BlobsSidecarGetter interface {
	BlobsSidecar(blockRoot [32]byte, index uint64) (*eth.BlobsSidecar, error)
}

type BlobsSidecarWriter interface {
	WriteBlobsSidecar(blockRoot [32]byte, index uint64, sidecar *eth.BlobsSidecar) error
}

type BlobsDB interface {
	BlobsSidecarGetter
	BlobsSidecarWriter
}
