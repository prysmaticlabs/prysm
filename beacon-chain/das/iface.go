package das

import (
	"context"

	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

// BlobsDB specifies the persistence store methods needed by the AvailabilityStore.
type BlobsDB interface {
	BlobSidecarsByRoot(ctx context.Context, beaconBlockRoot [32]byte, indices ...uint64) ([]*ethpb.BlobSidecar, error)
	SaveBlobSidecar(ctx context.Context, sidecars []*ethpb.BlobSidecar) error
}

// AvailabilityStore describes a component that can verify and save sidecars for a given block, and confirm previously
// verified and saved sidecars.
type AvailabilityStore interface {
	IsDataAvailable(ctx context.Context, current primitives.Slot, b blocks.ROBlock) error
	PersistOnceCommitted(ctx context.Context, current primitives.Slot, sc ...*ethpb.BlobSidecar) []*ethpb.BlobSidecar
}
