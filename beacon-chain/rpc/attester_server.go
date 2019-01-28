package rpc

import (
	"context"
	"fmt"
	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

type AttesterServer struct {
	attestationService    attestationService
}

// AttestHead is a function called by an attester in a sharding validator to vote
// on a block.
func (as *AttesterServer) AttestHead(ctx context.Context, req *pb.AttestRequest) (*pb.AttestResponse, error) {
	enc, err := proto.Marshal(req.Attestation)
	if err != nil {
		return nil, fmt.Errorf("could not marshal attestation: %v", err)
	}
	h := hashutil.Hash(enc)
	// Relays the attestation to chain service.
	as.attestationService.IncomingAttestationFeed().Send(req.Attestation)

	return &pb.AttestResponse{AttestationHash: h[:]}, nil
}
