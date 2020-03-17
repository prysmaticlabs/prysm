package iface

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

// SpanDetector defines an interface for Spanners to follow to allow mocks.
type ProposalsDetector interface {
	DetectDoublePropose(ctx context.Context, incomingBlk *ethpb.SignedBeaconBlockHeader) (*ethpb.ProposerSlashing, error)
}
