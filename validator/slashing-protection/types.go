package slashingprotection

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
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

// AttestingHistoryManager defines a struct which is able
// to perform different methods to retrieve and persist attesting history.
type AttestingHistoryManager interface {
	SaveAttestingHistoryForPubKey(ctx context.Context, pubKey [48]byte) error
	LoadAttestingHistoryForPubKeys(ctx context.Context, attestingPubKeys [][48]byte) error
	AttestingHistoryForPubKey(ctx context.Context, pubKey [48]byte) (kv.EncHistoryData, error)
	ResetAttestingHistoryForEpoch(ctx context.Context)
}
