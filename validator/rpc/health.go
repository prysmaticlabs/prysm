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
	filteredIndices := s.filteredIndices(ctx, req)
	returnedKeys := make([][]byte, 0, len(filteredIndices))
	returnedIndices := make([]uint64, 0, len(filteredIndices))
	returnedBalances := make([]uint64, 0, len(filteredIndices))

	indices := s.validatorService.ValidatorIndicesToPubkeys(ctx)
	balances := s.validatorService.ValidatorBalances(ctx)
	for _, i := range filteredIndices {
		k, ok := indices[i]
		if !ok {
			continue
		}
		b, ok := balances[k]
		if !ok {
			continue
		}
		returnedKeys = append(returnedKeys, k[:])
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
	filteredIndices := s.filteredIndices(ctx, req)
	returnedKeys := make([][]byte, 0, len(filteredIndices))
	returnedIndices := make([]uint64, 0, len(filteredIndices))
	returnedStatuses := make([]pb.ListStatusesResponse_ValidatorStatus, 0, len(filteredIndices))

	indices := s.validatorService.ValidatorIndicesToPubkeys(ctx)
	statuses := s.validatorService.ValidatorPubkeysToStatuses(ctx)
	for _, i := range filteredIndices {
		k, ok := indices[i]
		if !ok {
			continue
		}
		s, ok := statuses[k]
		if !ok {
			continue
		}
		returnedKeys = append(returnedKeys, k[:])
		returnedStatuses = append(returnedStatuses, pb.ListStatusesResponse_ValidatorStatus(s))
		returnedIndices = append(returnedIndices, i)
	}

	return &pb.ListStatusesResponse{
		PublicKeys: returnedKeys,
		Indices:    returnedIndices,
		Statuses:   returnedStatuses,
	}, nil
}

// ListPerformance lists the validator current performances.
func (s *Server) ListPerformance(ctx context.Context, req *pb.AccountRequest) (*pb.ListPerformanceResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s *Server) filteredIndices(ctx context.Context, req *pb.AccountRequest) []uint64 {
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
	return filteredIndices
}
