package rpc

import (
	"context"

	ptypes "github.com/gogo/protobuf/types"
	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreateAccount allows the user to create a new validator
// account using their wallet.
func (s *Server) CreateAccount(ctx context.Context, _ *ptypes.Empty) (*pb.CreateAccountResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

// ListAccounts allows us to retrieve a list of validator accounts
// currently managed by the user's wallet.
func (s *Server) ListAccounts(ctx context.Context, req *pb.ListAccountsRequest) (*pb.ListAccountsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}
