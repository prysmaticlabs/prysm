// Package iface exists to prevent circular dependencies when implementing the database interface.
package iface

import (
	"context"
	"io"

	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
)

// ValidatorDB defines the necessary methods for a Prysm validator DB.
type ValidatorDB interface {
	io.Closer
	DatabasePath() string
	ClearDB() error
	// Proposer protection related methods.
	ProposalHistory(ctx context.Context, publicKey []byte) (*slashpb.ProposalHistory, error)
	SaveProposalHistory(ctx context.Context, publicKey []byte, history *slashpb.ProposalHistory) error
	DeleteProposalHistory(ctx context.Context, publicKey []byte) error
	// Attester protection related methods.
	AttestationHistory(ctx context.Context, publicKey []byte) (*slashpb.AttestationHistory, error)
	SaveAttestationHistory(ctx context.Context, publicKey []byte, history *slashpb.AttestationHistory) error
	DeleteAttestationHistory(ctx context.Context, publicKey []byte) error
}
