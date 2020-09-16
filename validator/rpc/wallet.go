package rpc

import (
	"context"
	"encoding/json"
	"path/filepath"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/rand"
	v2 "github.com/prysmaticlabs/prysm/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/validator/accounts/v2/wallet"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/tyler-smith/go-bip39"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var defaultWalletPath = filepath.Join(flags.DefaultValidatorDir(), flags.WalletDefaultDirName)

// CreateWallet via an API request, allowing a user to save a new
// derived, direct, or remote wallet.
func (s *Server) CreateWallet(ctx context.Context, req *pb.CreateWalletRequest) (*pb.WalletResponse, error) {
	switch req.Keymanager {
	case pb.CreateWalletRequest_DIRECT:
		// Needs to unmarshal the keystores from the requests.
		if req.KeystoresImported == nil || len(req.KeystoresImported) < 1 {
			return nil, status.Error(codes.InvalidArgument, "No keystores included for import")
		}
		keystores := make([]*v2keymanager.Keystore, len(req.KeystoresImported))
		for i := 0; i < len(req.KeystoresImported); i++ {
			encoded := req.KeystoresImported[i]
			keystore := &v2keymanager.Keystore{}
			if err := json.Unmarshal([]byte(encoded), &keystore); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "Not a valid EIP-2335 keystore JSON file: %v", err)
			}
			keystores[i] = keystore
		}
		w, err := v2.CreateWalletWithKeymanager(ctx, &v2.CreateWalletConfig{
			WalletCfg: &wallet.Config{
				WalletDir:      defaultWalletPath,
				KeymanagerKind: v2keymanager.Direct,
				WalletPassword: req.WalletPassword,
			},
			SkipMnemonicConfirm: true,
		})
		if err != nil {
			return nil, err
		}
		// Import the uploaded accounts.
		if err := v2.ImportAccounts(ctx, &v2.ImportAccountsConfig{
			Wallet:          w,
			Keystores:       keystores,
			AccountPassword: req.KeystoresPassword,
		}); err != nil {
			return nil, err
		}
		return &pb.WalletResponse{
			WalletPath: defaultWalletPath,
		}, nil
	case pb.CreateWalletRequest_DERIVED:
		if req.NumAccounts < 1 {
			return nil, status.Error(codes.InvalidArgument, "Must create at least 1 validator account")
		}
		if req.Mnemonic == "" {
			return nil, status.Error(codes.InvalidArgument, "Must include mnemonic in request")
		}
		_, err := v2.RecoverWallet(ctx, &v2.RecoverWalletConfig{
			WalletDir:      defaultWalletPath,
			WalletPassword: req.WalletPassword,
			Mnemonic:       req.Mnemonic,
			NumAccounts:    int64(req.NumAccounts),
		})
		if err != nil {
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
}

// EditConfig allows the user to edit their wallet's keymanageropts.
func (s *Server) EditConfig(ctx context.Context, req *pb.EditWalletConfigRequest) (*pb.WalletResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

// WalletConfig returns the wallet's configuration. If no wallet exists, we return an empty response.
func (s *Server) WalletConfig(ctx context.Context, _ *ptypes.Empty) (*pb.WalletResponse, error) {
	err := wallet.Exists(defaultWalletPath)
	if err != nil && errors.Is(err, wallet.ErrNoWalletFound) {
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

// GenerateMnemonic creates a new, random bip39 mnemonic phrase.
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
