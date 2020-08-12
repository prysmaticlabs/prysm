package rpc

import (
	"context"
	"sort"

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

	indices := s.validatorService.ValidatorIndicesToPubkeys(ctx)
	for _, i := range req.Indices {
		pb, ok := indices[i]
		if ok {
			filtered[pb] = true
		}
	}

	pubkeys := s.validatorService.ValidatorPubkeysToIndices(ctx)
	balances := s.validatorService.ValidatorBalances(ctx)
	returnedKeys := make([][]byte, 0, len(filtered))
	returnedIndices := make([]uint64, 0, len(filtered))
	returnedBalances := make([]uint64, 0, len(filtered))

	filteredIndices := make([]uint64, 0, len(filtered))
	for k := range filtered {
		i, ok := pubkeys[k]
		if ok {
			filteredIndices = append(filteredIndices, i)
		}
	}
	sort.Slice(filteredIndices, func(i int, j int) bool {
		return filteredIndices[i] < filteredIndices[j]
	})

	for _, i := range filteredIndices {
		k := indices[i]
		returnedKeys = append(returnedKeys, k[:])
		b := balances[k]
		returnedBalances = append(returnedBalances, b)
		returnedIndices = append(returnedIndices, i)
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
