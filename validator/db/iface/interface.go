// Package iface defines an interface for the validator database.
package iface

import (
	"context"
	"io"

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

	// Attester protection related methods.
	LowestSignedTargetEpoch(ctx context.Context, publicKey [48]byte) (uint64, error)
	LowestSignedSourceEpoch(ctx context.Context, publicKey [48]byte) (uint64, error)
	SaveLowestSignedTargetEpoch(ctx context.Context, publicKey [48]byte, epoch uint64) error
	SaveLowestSignedSourceEpoch(ctx context.Context, publicKey [48]byte, epoch uint64) error
	AttestationHistoryForPubKeysV2(ctx context.Context, publicKeys [][48]byte) (map[[48]byte]kv.EncHistoryData, error)
	SaveAttestationHistoryForPubKeysV2(ctx context.Context, historyByPubKeys map[[48]byte]kv.EncHistoryData) error
	SaveAttestationHistoryForPubKeyV2(ctx context.Context, pubKey [48]byte, history kv.EncHistoryData) error
	AttestedPublicKeys(ctx context.Context) ([][48]byte, error)
}
