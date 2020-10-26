package rpc

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/pagination"
	"github.com/prysmaticlabs/prysm/shared/petnames"
	"github.com/prysmaticlabs/prysm/validator/accounts"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/derived"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type accountCreator interface {
	CreateAccount(ctx context.Context) ([]byte, *ethpb.Deposit_Data, error)
}

// CreateAccount allows creation of a new account in a user's wallet via RPC.
func (s *Server) CreateAccount(ctx context.Context, req *pb.CreateAccountRequest) (*pb.DepositDataResponse, error) {
	if !s.walletInitialized {
		return nil, status.Error(codes.FailedPrecondition, "Wallet not yet initialized")
	}
	km, ok := s.keymanager.(*derived.Keymanager)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "Only HD wallets can create accounts")
	}
	dataList := make([]*pb.DepositDataResponse_DepositData, req.NumAccounts)
	for i := uint64(0); i < req.NumAccounts; i++ {
		data, err := createAccountWithDepositData(ctx, km)
		if err != nil {
			return nil, err
		}
		dataList[i] = data
	}
	return &pb.DepositDataResponse{
		DepositDataList: dataList,
	}, nil
}

// ListAccounts allows retrieval of validating keys and their petnames
// for a user's wallet via RPC.
func (s *Server) ListAccounts(ctx context.Context, req *pb.ListAccountsRequest) (*pb.ListAccountsResponse, error) {
	if !s.walletInitialized {
		return nil, status.Error(codes.FailedPrecondition, "Wallet not yet initialized")
	}
	if int(req.PageSize) > cmd.Get().MaxRPCPageSize {
		return nil, status.Errorf(codes.InvalidArgument, "Requested page size %d can not be greater than max size %d",
			req.PageSize, cmd.Get().MaxRPCPageSize)
	}
	keys, err := s.keymanager.FetchValidatingPublicKeys(ctx)
	if err != nil {
		return nil, err
	}
	accs := make([]*pb.Account, len(keys))
	for i := 0; i < len(keys); i++ {
		accs[i] = &pb.Account{
			ValidatingPublicKey: keys[i][:],
			AccountName:         petnames.DeterministicName(keys[i][:], "-"),
		}
		if s.wallet.KeymanagerKind() == keymanager.Derived {
			accs[i].DerivationPath = fmt.Sprintf(derived.ValidatingKeyDerivationPathTemplate, i)
		}
	}
	if req.All {
		return &pb.ListAccountsResponse{
			Accounts:      accs,
			TotalSize:     int32(len(keys)),
			NextPageToken: "",
		}, nil
	}
	start, end, nextPageToken, err := pagination.StartAndEndPage(req.PageToken, int(req.PageSize), len(keys))
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			"Could not paginate results: %v",
			err,
		)
	}
	return &pb.ListAccountsResponse{
		Accounts:      accs[start:end],
		TotalSize:     int32(len(keys)),
		NextPageToken: nextPageToken,
	}, nil
}

// DeleteAccounts deletes accounts from a user if their wallet is an imported wallet.
func (s *Server) DeleteAccounts(
	ctx context.Context, req *pb.DeleteAccountsRequest,
) (*pb.DeleteAccountsResponse, error) {
	if req.PublicKeys == nil || len(req.PublicKeys) < 1 {
		return nil, status.Error(codes.InvalidArgument, "No public keys specified to delete")
	}
	if s.wallet == nil || s.keymanager == nil {
		return nil, status.Error(codes.FailedPrecondition, "No wallet found")
	}
	if s.wallet.KeymanagerKind() != keymanager.Imported {
		return nil, status.Error(codes.FailedPrecondition, "Only imported wallets can delete accounts")
	}
	if err := accounts.DeleteAccount(ctx, &accounts.DeleteAccountConfig{
		Wallet:     s.wallet,
		Keymanager: s.keymanager,
		PublicKeys: req.PublicKeys,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not delete public keys: %v", err)
	}
	return &pb.DeleteAccountsResponse{
		DeletedKeys: req.PublicKeys,
	}, nil
}

func createAccountWithDepositData(ctx context.Context, km accountCreator) (*pb.DepositDataResponse_DepositData, error) {
	// Create a new validator account using the specified keymanager.
	_, depositData, err := km.CreateAccount(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not create account in wallet")
	}
	data, err := accounts.DepositDataJSON(depositData)
	if err != nil {
		return nil, errors.Wrap(err, "could not create deposit data JSON")
	}
	return &pb.DepositDataResponse_DepositData{
		Data: data,
	}, nil
}
