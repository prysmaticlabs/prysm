// Package iface defines an interface for a double-proposal detector struct.
package iface

import (
	"context"

	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// ProposalsDetector defines an interface for different implementations.
type ProposalsDetector interface {
	DetectDoublePropose(ctx context.Context, incomingBlk *ethpb.SignedBeaconBlockHeader) (*ethpb.ProposerSlashing, error)
	DetectDoubleProposeNoUpdate(ctx context.Context, incomingBlk *ethpb.BeaconBlockHeader) (bool, error)
}
