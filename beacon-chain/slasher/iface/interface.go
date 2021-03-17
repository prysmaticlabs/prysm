// Package iface defines the interface for interacting with the slasher,
// in order to detect slashings.
package iface

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

// SlashableChecker is an interface for defining services that the beacon node may interact with to provide slashing data.
type SlashableChecker interface {
	IsSlashableProposal(ctx context.Context, proposal *ethpb.SignedBeaconBlockHeader) (*ethpb.ProposerSlashing, error)
	IsSlashableAttestation(ctx context.Context, attestation *ethpb.IndexedAttestation) ([]*ethpb.AttesterSlashing, error)
}
