// Package iface defines an interface for the validator database.
package iface

import (
	"context"
	"io"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/backuputil"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
)

// Ensure the kv store implements the interface.
var _ = ValidatorDB(&kv.Store{})

// ValidatorDB defines the necessary methods for a Prysm validator DB.
type ValidatorDB interface {
	io.Closer
	backuputil.BackupExporter
	DatabasePath() string
	ClearDB() error
	UpdatePublicKeysBuckets(publicKeys [][48]byte) error

	// Genesis information related methods.
	GenesisValidatorsRoot(ctx context.Context) ([]byte, error)
	SaveGenesisValidatorsRoot(ctx context.Context, genValRoot []byte) error

	// Proposer protection related methods.
	HighestSignedProposal(ctx context.Context, publicKey [48]byte) (uint64, error)
	LowestSignedProposal(ctx context.Context, publicKey [48]byte) (uint64, error)
	ProposalHistoryForSlot(ctx context.Context, publicKey [48]byte, slot uint64) ([32]byte, bool, error)
	SaveProposalHistoryForSlot(ctx context.Context, pubKey [48]byte, slot uint64, signingRoot []byte) error
	ProposedPublicKeys(ctx context.Context) ([][48]byte, error)

	// Optimal attester protection related methods.
	LowestSignedTargetEpoch(ctx context.Context, publicKey [48]byte) (uint64, error)
	LowestSignedSourceEpoch(ctx context.Context, publicKey [48]byte) (uint64, error)
	SaveLowestSignedTargetEpoch(ctx context.Context, publicKey [48]byte, epoch uint64) error
	SaveLowestSignedSourceEpoch(ctx context.Context, publicKey [48]byte, epoch uint64) error
	AttestedPublicKeys(ctx context.Context) ([][48]byte, error)
	CheckSlashableAttestation(
		ctx context.Context, pubKey [48]byte, signingRoot [32]byte, att *ethpb.IndexedAttestation,
	) (kv.SlashingKind, error)
	ApplyAttestationForPubKey(
		ctx context.Context, pubKey [48]byte, signingRoot [32]byte, att *ethpb.IndexedAttestation,
	) error
}
