package rpc

import (
	"context"
	"fmt"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/petnames"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreateAccount allows creation of a new account in a user's wallet via RPC.
func (s *Server) CreateAccount(ctx context.Context, _ *ptypes.Empty) (*pb.CreateAccountResponse, error) {
	if !s.walletInitialized {
		return nil, status.Error(codes.FailedPrecondition, "Wallet not yet initialized")
	}
	var pubKey []byte
	var err error
	switch s.wallet.KeymanagerKind() {
	case v2keymanager.Remote:
		return nil, status.Error(codes.InvalidArgument, "Cannot create account for remote keymanager")
	case v2keymanager.Direct:
		km, ok := s.keymanager.(*direct.Keymanager)
		if !ok {
			return nil, status.Error(codes.InvalidArgument, "Not a direct keymanager")
		}
		// Create a new validator account using the specified keymanager.
		pubKey, err = km.CreateAccount(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not create account in wallet")
		}
	case v2keymanager.Derived:
		km, ok := s.keymanager.(*derived.Keymanager)
		if !ok {
			return nil, status.Error(codes.InvalidArgument, "Not a derived keymanager")
		}
		pubKey, err = km.CreateAccount(ctx, false /*logAccountInfo*/)
		if err != nil {
			return nil, errors.Wrap(err, "could not create account in wallet")
		}
	}
	return &pb.CreateAccountResponse{
		Account: &pb.Account{
			ValidatingPublicKey: pubKey,
			AccountName:         petnames.DeterministicName(pubKey, "-"),
		},
	}, nil
}

// ListAccounts allows retrieval of validating keys and their petnames
// for a user's wallet via RPC.
func (s *Server) ListAccounts(ctx context.Context, req *pb.ListAccountsRequest) (*pb.ListAccountsResponse, error) {
	if !s.walletInitialized {
		return nil, status.Error(codes.FailedPrecondition, "Wallet not yet initialized")
	}
	keys, err := s.keymanager.FetchValidatingPublicKeys(ctx)
	if err != nil {
		return nil, err
	}
	accounts := make([]*pb.Account, len(keys))
	for i := 0; i < len(keys); i++ {
		accounts[i] = &pb.Account{
			ValidatingPublicKey: keys[i][:],
			AccountName:         petnames.DeterministicName(keys[i][:], "-"),
		}
		if s.wallet.KeymanagerKind() == v2keymanager.Derived {
			accounts[i].DerivationPath = fmt.Sprintf(derived.ValidatingKeyDerivationPathTemplate, i)
		}
	}
	return &pb.ListAccountsResponse{
		Accounts: accounts,
	}, nil
}
