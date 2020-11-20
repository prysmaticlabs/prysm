package slashingprotection

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared"
)

// Protector interface defines a struct which provides methods
// for validator slashing protection.
type Protector interface {
	IsSlashableAttestation(
		ctx context.Context,
		indexedAtt *ethpb.IndexedAttestation,
		pubKey [48]byte,
		signingRoot [32]byte,
	) (bool, error)
	IsSlashableBlock(
		ctx context.Context, block *ethpb.SignedBeaconBlock, pubKey [48]byte, signingRoot [32]byte,
	) (bool, error)
	shared.Service
}
