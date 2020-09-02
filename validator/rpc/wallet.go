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
	"github.com/tyler-smith/go-bip39"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreateWallet --
func (s *Server) CreateWallet(ctx context.Context, req *pb.CreateWalletRequest) (*pb.WalletResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
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
