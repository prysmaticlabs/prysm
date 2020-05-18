package slashingprotection

import (
	"context"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

// Notifier interface defines the methods of the service that provides block updates to consumers.
type Protector interface {
	VerifyAttestation(ctx context.Context, attestation *eth.IndexedAttestation) bool
	VerifyBlock(ctx context.Context, blockHeader *eth.SignedBeaconBlockHeader) bool
}
