package rpc

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/rand"
	v2 "github.com/prysmaticlabs/prysm/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/validator/accounts/v2/wallet"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	"github.com/tyler-smith/go-bip39"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var defaultWalletPath = filepath.Join(flags.DefaultValidatorDir(), flags.WalletDefaultDirName)

// HasWallet checks if a user has created a wallet before as well as whether or not
// they have used the web UI before to set a wallet password.
func (s *Server) HasWallet(ctx context.Context, _ *ptypes.Empty) (*pb.HasWalletResponse, error) {
	err := wallet.Exists(defaultWalletPath)
	if err != nil && errors.Is(err, wallet.ErrNoWalletFound) {
		return &pb.HasWalletResponse{
			WalletExists: false,
		}, nil
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not check if wallet exists: %v", err)
	}
	return &pb.HasWalletResponse{
		WalletExists: true,
	}, nil
}

// CreateWallet via an API request, allowing a user to save a new
// derived, direct, or remote wallet.
func (s *Server) CreateWallet(ctx context.Context, req *pb.CreateWalletRequest) (*pb.WalletResponse, error) {
	switch req.Keymanager {
	case pb.KeymanagerKind_DIRECT:
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
		if err := s.initializeWallet(ctx, &wallet.Config{
			WalletDir:      defaultWalletPath,
			KeymanagerKind: v2keymanager.Direct,
			WalletPassword: req.WalletPassword,
		}); err != nil {
			return nil, err
		}
		return &pb.WalletResponse{
			WalletPath: defaultWalletPath,
		}, nil
	case pb.KeymanagerKind_DERIVED:
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
		if err := s.initializeWallet(ctx, &wallet.Config{
			WalletDir:      defaultWalletPath,
			KeymanagerKind: v2keymanager.Direct,
			WalletPassword: req.WalletPassword,
		}); err != nil {
			return nil, err
		}
		return &pb.WalletResponse{
			WalletPath: defaultWalletPath,
		}, nil
	case pb.KeymanagerKind_REMOTE:
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
	if s.wallet == nil || s.keymanager == nil {
		// If no wallet is found, we simply return an empty response.
		return &pb.WalletResponse{}, nil
	}
	var keymanagerKind pb.KeymanagerKind
	switch s.wallet.KeymanagerKind() {
	case v2keymanager.Derived:
		keymanagerKind = pb.KeymanagerKind_DERIVED
	case v2keymanager.Direct:
		keymanagerKind = pb.KeymanagerKind_DIRECT
	case v2keymanager.Remote:
		keymanagerKind = pb.KeymanagerKind_REMOTE
	}
	f, err := s.wallet.ReadKeymanagerConfigFromDisk(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not read keymanager config from disk: %v", err)
	}
	encoded, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not parse keymanager config: %v", err)
	}
	var config map[string]string
	if err := json.Unmarshal(encoded, &config); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not JSON unmarshal keymanager config: %v", err)
	}
	return &pb.WalletResponse{
		WalletPath:       defaultWalletPath,
		KeymanagerKind:   keymanagerKind,
		KeymanagerConfig: config,
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

// ChangePassword allows changing a wallet password via the API as
// an authenticated method.
func (s *Server) ChangePassword(ctx context.Context, req *pb.ChangePasswordRequest) (*ptypes.Empty, error) {
	err := wallet.Exists(defaultWalletPath)
	if err != nil && errors.Is(err, wallet.ErrNoWalletFound) {
		return nil, status.Error(codes.FailedPrecondition, "No wallet found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not check if wallet exists: %v", err)
	}
	if req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "Password cannot be empty")
	}
	if req.Password != req.PasswordConfirmation {
		return nil, status.Error(codes.InvalidArgument, "Password does not match confirmation")
	}
	switch s.wallet.KeymanagerKind() {
	case v2keymanager.Direct:
		km, ok := s.keymanager.(*direct.Keymanager)
		if !ok {
			return nil, status.Error(codes.FailedPrecondition, "Not a valid direct keymanager")
		}
		s.wallet.SetPassword(req.Password)
		if err := s.wallet.SaveHashedPassword(ctx); err != nil {
			return nil, status.Errorf(codes.Internal, "Could not save hashed password: %v", err)
		}
		if err := km.RefreshWalletPassword(ctx); err != nil {
			return nil, status.Errorf(codes.Internal, "Could not refresh wallet password: %v", err)
		}
	case v2keymanager.Derived:
		km, ok := s.keymanager.(*derived.Keymanager)
		if !ok {
			return nil, status.Error(codes.FailedPrecondition, "Not a valid derived keymanager")
		}
		s.wallet.SetPassword(req.Password)
		if err := s.wallet.SaveHashedPassword(ctx); err != nil {
			return nil, status.Errorf(codes.Internal, "Could not save hashed password: %v", err)
		}
		if err := km.RefreshWalletPassword(ctx); err != nil {
			return nil, status.Errorf(codes.Internal, "Could not refresh wallet password: %v", err)
		}
	case v2keymanager.Remote:
		return nil, status.Error(codes.Internal, "Cannot change password for remote keymanager")
	}
	return &ptypes.Empty{}, nil
}
