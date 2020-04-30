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
	DeleteProposalHistory(ctx context.Context, publicKey []byte) error
	// Attester protection related methods.
	AttestationHistory(ctx context.Context, publicKey []byte) (*slashpb.AttestationHistory, error)
	SaveAttestationHistory(ctx context.Context, publicKey []byte, history *slashpb.AttestationHistory) error
	DeleteAttestationHistory(ctx context.Context, publicKey []byte) error
}
