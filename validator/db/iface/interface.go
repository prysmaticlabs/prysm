// Package iface defines an interface for the validator database.
package iface

import (
	"context"
	"io"

	"github.com/prysmaticlabs/prysm/validator/db/kv"
	attestinghistory "github.com/prysmaticlabs/prysm/validator/slashing-protection/local/attesting-history"
)

// ValidatorDB defines the necessary methods for a Prysm validator DB.
type ValidatorDB interface {
	io.Closer
	DatabasePath() string
	ClearDB() error
	UpdatePublicKeysBuckets(publicKeys [][48]byte) error

	// Genesis information related methods.
	GenesisValidatorsRoot(ctx context.Context) ([]byte, error)
	SaveGenesisValidatorsRoot(ctx context.Context, genValRoot []byte) error

	// Proposal history methods for slashing protection.
	ProposalHistoryForSlot(ctx context.Context, publicKey []byte, slot uint64) ([]byte, error)
	SaveProposalHistoryForSlot(ctx context.Context, pubKey []byte, slot uint64, signingRoot []byte) error
	SaveProposalHistoryForPubKeys(ctx context.Context, proposals map[[48]byte]kv.ProposalHistoryForPubkey) error

	// Attesting history methods for slashing protection.
	AttestationHistoryForPubKeys(ctx context.Context, publicKeys [][48]byte) (map[[48]byte]attestinghistory.History, error)
	AttestationHistoryForPubKey(ctx context.Context, publicKey [48]byte) (attestinghistory.History, error)
	SaveAttestationHistoryForPubKeys(ctx context.Context, historyByPubKeys map[[48]byte]attestinghistory.History) error
	SaveAttestationHistoryForPubKey(ctx context.Context, pubKey [48]byte, history attestinghistory.History) error
}
