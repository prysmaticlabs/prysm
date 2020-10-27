// Package iface defines an interface for the validator database.
package iface

import (
	"context"
	"io"

	types "github.com/farazdagi/prysm-shared-types"
	"github.com/prysmaticlabs/go-bitfield"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
)

// ValidatorDB defines the necessary methods for a Prysm validator DB.
type ValidatorDB interface {
	io.Closer
	DatabasePath() string
	ClearDB() error
	UpdatePublicKeysBuckets(publicKeys [][48]byte) error
	// Proposer protection related methods.
	ProposalHistoryForEpoch(ctx context.Context, publicKey []byte, epoch types.Epoch) (bitfield.Bitlist, error)
	SaveProposalHistoryForEpoch(ctx context.Context, publicKey []byte, epoch types.Epoch, history bitfield.Bitlist) error
	//new data structure methods
	ProposalHistoryForSlot(ctx context.Context, publicKey []byte, slot types.Slot) ([]byte, error)
	SaveProposalHistoryForSlot(ctx context.Context, pubKey []byte, slot types.Slot, signingRoot []byte) error

	// Attester protection related methods.
	AttestationHistoryForPubKeys(ctx context.Context, publicKeys [][48]byte) (map[[48]byte]*slashpb.AttestationHistory, error)
	SaveAttestationHistoryForPubKeys(ctx context.Context, historyByPubKey map[[48]byte]*slashpb.AttestationHistory) error
}
