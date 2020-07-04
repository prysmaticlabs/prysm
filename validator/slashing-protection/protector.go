package slashingprotection

import (
	"context"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

// Protector interface defines the methods of the service that provides slashing protection.
type Protector interface {
	CheckAttestationSafety(ctx context.Context, attestation *eth.IndexedAttestation) bool
	CommitAttestation(ctx context.Context, attestation *eth.IndexedAttestation) bool
	CheckBlockSafety(ctx context.Context, blockHeader *eth.BeaconBlockHeader) bool
	CommitBlock(ctx context.Context, blockHeader *eth.SignedBeaconBlockHeader) bool
}
