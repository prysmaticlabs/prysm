// Package iface defines an interface for the validator database.
package iface

import (
	"context"
	"io"

	types "github.com/prysmaticlabs/eth2-types"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/monitoring/backup"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
)

// Ensure the kv store implements the interface.
var _ = ValidatorDB(&kv.Store{})

// ValidatorDB defines the necessary methods for a Prysm validator DB.
type ValidatorDB interface {
	io.Closer
	backup.BackupExporter
	DatabasePath() string
	ClearDB() error
	RunUpMigrations(ctx context.Context) error
	RunDownMigrations(ctx context.Context) error
	UpdatePublicKeysBuckets(publicKeys [][fieldparams.BLSPubkeyLength]byte) error

	// Genesis information related methods.
	GenesisValidatorsRoot(ctx context.Context) ([]byte, error)
	SaveGenesisValidatorsRoot(ctx context.Context, genValRoot []byte) error

	// Proposer protection related methods.
	HighestSignedProposal(ctx context.Context, publicKey [fieldparams.BLSPubkeyLength]byte) (types.Slot, bool, error)
	LowestSignedProposal(ctx context.Context, publicKey [fieldparams.BLSPubkeyLength]byte) (types.Slot, bool, error)
	ProposalHistoryForPubKey(ctx context.Context, publicKey [fieldparams.BLSPubkeyLength]byte) ([]*kv.Proposal, error)
	ProposalHistoryForSlot(ctx context.Context, publicKey [fieldparams.BLSPubkeyLength]byte, slot types.Slot) ([32]byte, bool, error)
	SaveProposalHistoryForSlot(ctx context.Context, pubKey [fieldparams.BLSPubkeyLength]byte, slot types.Slot, signingRoot []byte) error
	ProposedPublicKeys(ctx context.Context) ([][fieldparams.BLSPubkeyLength]byte, error)

	// Attester protection related methods.
	// Methods to store and read blacklisted public keys from EIP-3076
	// slashing protection imports.
	EIPImportBlacklistedPublicKeys(ctx context.Context) ([][fieldparams.BLSPubkeyLength]byte, error)
	SaveEIPImportBlacklistedPublicKeys(ctx context.Context, publicKeys [][fieldparams.BLSPubkeyLength]byte) error
	SigningRootAtTargetEpoch(ctx context.Context, publicKey [fieldparams.BLSPubkeyLength]byte, target types.Epoch) ([32]byte, error)
	LowestSignedTargetEpoch(ctx context.Context, publicKey [fieldparams.BLSPubkeyLength]byte) (types.Epoch, bool, error)
	LowestSignedSourceEpoch(ctx context.Context, publicKey [fieldparams.BLSPubkeyLength]byte) (types.Epoch, bool, error)
	AttestedPublicKeys(ctx context.Context) ([][fieldparams.BLSPubkeyLength]byte, error)
	CheckSlashableAttestation(
		ctx context.Context, pubKey [fieldparams.BLSPubkeyLength]byte, signingRoot [32]byte, att *ethpb.IndexedAttestation,
	) (kv.SlashingKind, error)
	SaveAttestationForPubKey(
		ctx context.Context, pubKey [fieldparams.BLSPubkeyLength]byte, signingRoot [32]byte, att *ethpb.IndexedAttestation,
	) error
	SaveAttestationsForPubKey(
		ctx context.Context, pubKey [fieldparams.BLSPubkeyLength]byte, signingRoots [][32]byte, atts []*ethpb.IndexedAttestation,
	) error
	AttestationHistoryForPubKey(
		ctx context.Context, pubKey [fieldparams.BLSPubkeyLength]byte,
	) ([]*kv.AttestationRecord, error)

	// Graffiti ordered index related methods
	SaveGraffitiOrderedIndex(ctx context.Context, index uint64) error
	GraffitiOrderedIndex(ctx context.Context, fileHash [32]byte) (uint64, error)
}
