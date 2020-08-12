package rpc

import (
	"context"

	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ListBalances lists the validator balances.
func (s *Server) ListBalances(ctx context.Context, req *pb.AccountRequest) (*pb.ListBalancesResponse, error) {
	filtered := map[[48]byte]bool{}
	for _, p := range req.PublicKeys {
		pb := bytesutil.ToBytes48(p)
		filtered[pb] = true
	}

	pubkeyToIndices := map[[48]byte]uint64{}
	for _, i := range req.Indices {
		indices := s.validatorService.ValidatorPubKeyToIndices(ctx)
		pb, ok := indices[i]
		if ok {
			filtered[pb] = true
			pubkeyToIndices[pb] = i
		}
	}

	balances := s.validatorService.ValidatorBalances(ctx)
	returnedKeys := make([][]byte, 0, len(filtered))
	returnedIndices := make([]uint64, 0, len(filtered))
	returnedBalances := make([]uint64, 0, len(filtered))
	for k := range filtered {
		b, ok := balances[k]
		if ok {
			returnedKeys = append(returnedKeys, k[:])
			returnedBalances = append(returnedBalances, b)
			if _, ok := pubkeyToIndices[k]; ok {
				returnedIndices = append(returnedIndices, pubkeyToIndices[k])
			}
		}
	}

	return &pb.ListBalancesResponse{
		PublicKeys: returnedKeys,
		Indices:    returnedIndices,
		Balances:   returnedBalances,
	}, nil
}

// ListStatuses lists the validator current statuses.
func (s *Server) ListStatuses(ctx context.Context, req *pb.AccountRequest) (*pb.ListStatusesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

// ListPerformance lists the validator current performances.
func (s *Server) ListPerformance(ctx context.Context, req *pb.AccountRequest) (*pb.ListPerformanceResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}
