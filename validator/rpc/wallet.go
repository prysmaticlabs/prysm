package rpc

import (
	"context"
	"path/filepath"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/rand"
	v2 "github.com/prysmaticlabs/prysm/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"

	"github.com/tyler-smith/go-bip39"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreateWallet --
func (s *Server) CreateWallet(ctx context.Context, req *pb.CreateWalletRequest) (*pb.WalletResponse, error) {
	if req.NumAccounts < 1 {
		return nil, status.Error(codes.InvalidArgument, "Must create at least 1 validator account")
	}
	defaultWalletPath := filepath.Join(flags.DefaultValidatorDir(), flags.WalletDefaultDirName)
	switch req.Keymanager {
	case pb.CreateWalletRequest_DIRECT:
	case pb.CreateWalletRequest_DERIVED:
		wallet, err := v2.CreateWalletWithKeymanager(ctx, &v2.CreateWalletConfig{
			WalletCfg: &v2.WalletConfig{
				WalletDir:      defaultWalletPath,
				KeymanagerKind: v2keymanager.Derived,
				WalletPassword: req.WalletPassword,
			},
			SkipMnemonicConfirm: true,
		})
		if err != nil {
			return nil, err
		}
		// Create the required accounts.
		if err := v2.CreateAccount(ctx, &v2.CreateAccountConfig{
			Wallet:      wallet,
			NumAccounts: int64(req.NumAccounts),
		}); err != nil {
			return nil, err
		}
		return &pb.WalletResponse{
			WalletPath: defaultWalletPath,
		}, nil
	case pb.CreateWalletRequest_REMOTE:
		return nil, status.Error(codes.Unimplemented, "Remote keymanager not yet supported")
	default:
		return nil, status.Errorf(codes.InvalidArgument, "Keymanager type %T not yet supported", req.Keymanager)
	}
	return nil, status.Error(codes.Internal, "could not select any keymanager")
}

// EditConfig --
func (s *Server) EditConfig(ctx context.Context, req *pb.EditWalletConfigRequest) (*pb.WalletResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

// WalletConfig --
func (s *Server) WalletConfig(ctx context.Context, _ *ptypes.Empty) (*pb.WalletResponse, error) {
	defaultWalletPath := filepath.Join(flags.DefaultValidatorDir(), flags.WalletDefaultDirName)
	err := v2.WalletExists(defaultWalletPath)
	if err != nil && errors.Is(err, v2.ErrNoWalletFound) {
		// If no wallet is found, we simply return an empty response.
		return &pb.WalletResponse{}, nil
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not check if wallet exists: %v", err)
	}
	return &pb.WalletResponse{
		WalletPath:       defaultWalletPath,
		KeymanagerConfig: nil, // Fill in by reading from disk.
	}, nil
}

// GenerateMnemonic --
func (s *Server) GenerateMnemonic(ctx context.Context, _ *ptypes.Empty) (*pb.GenerateMnemonicResponse, error) {
	mnemonicRandomness := make([]byte, 32)
	if _, err := rand.NewGenerator().Read(mnemonicRandomness); err != nil {
		return nil, status.Errorf(
			codes.FailedPrecondition,
			"Could not initialize mnemonic source of randomness: %v",
			err,
		)
	}
	mnemonic, err := bip39.NewMnemonic(mnemonicRandomness)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not generate wallet seed: %v", err)
	}
	return &pb.GenerateMnemonicResponse{
		Mnemonic: mnemonic,
	}, nil
}
