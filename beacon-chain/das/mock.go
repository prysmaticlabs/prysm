package das

import (
	"context"

	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

type MockAvailabilityStore struct {
	VerifyAvailabilityCallback func(ctx context.Context, current primitives.Slot, b blocks.ROBlock) error
	PersistBlobsCallback       func(ctx context.Context, current primitives.Slot, sc ...*ethpb.BlobSidecar) []*ethpb.BlobSidecar
}

var _ AvailabilityStore = &MockAvailabilityStore{}

func (m *MockAvailabilityStore) IsDataAvailable(ctx context.Context, current primitives.Slot, b blocks.ROBlock) error {
	if m.VerifyAvailabilityCallback != nil {
		return m.VerifyAvailabilityCallback(ctx, current, b)
	}
	return nil
}

func (m *MockAvailabilityStore) PersistOnceCommitted(ctx context.Context, current primitives.Slot, sc ...*ethpb.BlobSidecar) []*ethpb.BlobSidecar {
	if m.PersistBlobsCallback != nil {
		return m.PersistBlobsCallback(ctx, current, sc...)
	}
	return sc
}

type mockBlobsDB struct {
	BlobSidecarsByRootCallback func(ctx context.Context, root [32]byte, indices ...uint64) ([]*ethpb.BlobSidecar, error)
	SaveBlobSidecarCallback    func(ctx context.Context, sidecars []*ethpb.BlobSidecar) error
}

var _ BlobsDB = &mockBlobsDB{}

func (b *mockBlobsDB) BlobSidecarsByRoot(ctx context.Context, root [32]byte, indices ...uint64) ([]*ethpb.BlobSidecar, error) {
	if b.BlobSidecarsByRootCallback != nil {
		return b.BlobSidecarsByRootCallback(ctx, root, indices...)
	}
	return nil, nil
}

func (b *mockBlobsDB) SaveBlobSidecar(ctx context.Context, sidecars []*ethpb.BlobSidecar) error {
	if b.SaveBlobSidecarCallback != nil {
		return b.SaveBlobSidecarCallback(ctx, sidecars)
	}
	return nil
}
