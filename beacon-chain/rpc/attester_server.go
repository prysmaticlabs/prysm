package rpc

import (
	"context"
	"fmt"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

// AttesterServer defines a server implementation of the gRPC Attester service,
// providing RPC methods for validators acting as attesters to broadcast votes on beacon blocks.
type AttesterServer struct {
	operationService operationService
}

// AttestHead is a function called by an attester in a sharding validator to vote
// on a block via an attestation object as defined in the Ethereum Serenity specification.
func (as *AttesterServer) AttestHead(ctx context.Context, req *pb.AttestRequest) (*pb.AttestResponse, error) {
	h, err := hashutil.HashProto(req.Attestation)
	if err != nil {
		return nil, fmt.Errorf("could not hash attestation: %v", err)
	}
	// Relays the attestation to chain service.
	as.operationService.IncomingAttFeed().Send(req.Attestation)
	return &pb.AttestResponse{AttestationHash: h[:]}, nil
}
