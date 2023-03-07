package sync

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/db"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

type mockSidecarId struct {
	r [32]byte
	i uint64
}

type MockBlobDB struct {
	storage map[mockSidecarId]*ethpb.BlobSidecar
}

func (m *MockBlobDB) BlobSidecar(r [32]byte, idx uint64) (*ethpb.BlobSidecar, error) {
	sc, ok := m.storage[mockSidecarId{r, idx}]
	if !ok {
		return nil, errors.Wrapf(db.ErrNotFound, "MockBlobsDB.storage does not contain blob: root=%#x, idx=%d", r, idx)
	}
	return sc, nil
}

func (m *MockBlobDB) WriteBlobSidecar(r [32]byte, idx uint64, s *ethpb.BlobSidecar) error {
	if m.storage == nil {
		m.storage = make(map[mockSidecarId]*ethpb.BlobSidecar)
	}
	m.storage[mockSidecarId{r, idx}] = s
	return nil
}

var _ BlobDB = &MockBlobDB{}
