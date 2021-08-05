package iface

import (
	"context"

	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// OldRemoteSlasher is deprecated and will be removed.
type OldRemoteSlasher interface {
	CheckAttestationSafety(ctx context.Context, attestation *eth.IndexedAttestation) bool
	CommitAttestation(ctx context.Context, attestation *eth.IndexedAttestation) bool
	CheckBlockSafety(ctx context.Context, blockHeader *eth.BeaconBlockHeader) bool
	CommitBlock(ctx context.Context, blockHeader *eth.SignedBeaconBlockHeader) (bool, error)
	Status() error
}
