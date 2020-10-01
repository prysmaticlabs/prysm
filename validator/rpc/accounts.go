package rpc

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/petnames"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
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
	var creator accountCreator
	switch s.wallet.KeymanagerKind() {
	case v2keymanager.Remote:
		return nil, status.Error(codes.InvalidArgument, "Cannot create account for remote keymanager")
	case v2keymanager.Direct:
		km, ok := s.keymanager.(*direct.Keymanager)
		if !ok {
			return nil, status.Error(codes.InvalidArgument, "Not a direct keymanager")
		}
		creator = km
	case v2keymanager.Derived:
		km, ok := s.keymanager.(*derived.Keymanager)
		if !ok {
			return nil, status.Error(codes.InvalidArgument, "Not a derived keymanager")
		}
		creator = km
	}
	dataList := make([]*pb.DepositDataResponse_DepositData, req.NumAccounts)
	for i := uint64(0); i < req.NumAccounts; i++ {
		data, err := createAccountWithDepositData(ctx, creator)
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

// BackupAccounts --
func (s *Server) BackupAccounts(
	ctx context.Context, req *pb.BackupAccountsRequest,
) (*pb.BackupAccountsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

// DeleteAccounts --
func (s *Server) DeleteAccounts(
	ctx context.Context, req *pb.DeleteAccountsRequest,
) (*pb.DeleteAccountsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func createAccountWithDepositData(ctx context.Context, km accountCreator) (*pb.DepositDataResponse_DepositData, error) {
	// Create a new validator account using the specified keymanager.
	pubKey, depositData, err := km.CreateAccount(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not create account in wallet")
	}
	depositMessage := &pb.DepositMessage{
		Pubkey:                pubKey,
		WithdrawalCredentials: depositData.WithdrawalCredentials,
		Amount:                depositData.Amount,
	}
	depositMessageRoot, err := depositMessage.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	depositDataRoot, err := depositData.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	data := make(map[string]string)
	data["pubkey"] = fmt.Sprintf("%x", pubKey)
	data["withdrawal_credentials"] = fmt.Sprintf("%x", depositData.WithdrawalCredentials)
	data["amount"] = fmt.Sprintf("%d", depositData.Amount)
	data["signature"] = fmt.Sprintf("%x", depositData.Signature)
	data["deposit_message_root"] = fmt.Sprintf("%x", depositMessageRoot)
	data["deposit_data_root"] = fmt.Sprintf("%x", depositDataRoot)
	data["fork_version"] = fmt.Sprintf("%x", params.BeaconConfig().GenesisForkVersion)
	return &pb.DepositDataResponse_DepositData{
		Data: data,
	}, nil
}
