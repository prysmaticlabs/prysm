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
