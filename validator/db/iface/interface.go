// Package iface defines an interface for the validator database.
package iface

import (
	"context"
	"io"

	"github.com/prysmaticlabs/go-bitfield"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
)

// ValidatorDB defines the necessary methods for a Prysm validator DB.
type ValidatorDB interface {
	io.Closer
	DatabasePath() string
	ClearDB() error
	// Proposer protection related methods.
	ProposalHistoriesForEpoch(ctx context.Context, publicKeys [][48]byte, epoch uint64) (map[[48]byte]bitfield.Bitlist, error)
	SaveProposalHistoriesForEpoch(ctx context.Context, epoch uint64, histories map[[48]byte]bitfield.Bitlist) error
	DeleteProposalHistory(ctx context.Context, publicKey []byte) error
	// Attester protection related methods.
	AttestationHistoryForPubKeys(ctx context.Context, publicKeys [][48]byte) (map[[48]byte]*slashpb.AttestationHistory, error)
	SaveAttestationHistoryForPubKeys(ctx context.Context, historyByPubKey map[[48]byte]*slashpb.AttestationHistory) error
	DeleteAttestationHistory(ctx context.Context, publicKey []byte) error
}
