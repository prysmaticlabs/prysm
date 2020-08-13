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
	ProposalHistoryForEpoch(ctx context.Context, publicKey []byte, epoch uint64) (bitfield.Bitlist, error)
	SaveProposalHistoryForEpoch(ctx context.Context, publicKey []byte, epoch uint64, history bitfield.Bitlist) error
	// Attester protection related methods.
	AttestationHistoryForPubKeys(ctx context.Context, publicKeys [][48]byte) (map[[48]byte]*slashpb.AttestationHistory, error)
	SaveAttestationHistoryForPubKeys(ctx context.Context, historyByPubKey map[[48]byte]*slashpb.AttestationHistory) error
	// Validator RPC authentication methods.
	SaveHashedPasswordForAPI(ctx context.Context, hashedPassword []byte) error
	HashedPasswordForAPI(ctx context.Context) ([]byte, error)
}
