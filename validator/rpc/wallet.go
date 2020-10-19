package rpc

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"path/filepath"
	"strings"

	ptypes "github.com/gogo/protobuf/types"
	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/prysmaticlabs/prysm/shared/promptutil"
	"github.com/prysmaticlabs/prysm/shared/rand"
	"github.com/prysmaticlabs/prysm/validator/accounts"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/imported"
	"github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	checkExistsErrMsg   = "Could not check if wallet exists"
	checkValidityErrMsg = "Could not check if wallet is valid"
	noWalletMsg         = "No wallet found at path"
	invalidWalletMsg    = "Directory does not contain a valid wallet"
)

// HasWallet checks if a user has created a wallet before as well as whether or not
// they have used the web UI before to set a wallet password.
func (s *Server) HasWallet(_ context.Context, _ *ptypes.Empty) (*pb.HasWalletResponse, error) {
	exists, err := wallet.Exists(s.walletDir)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not check if wallet exists: %v", err)
	}
	if !exists {
		return &pb.HasWalletResponse{
			WalletExists: false,
		}, nil
	}
	return &pb.HasWalletResponse{
		WalletExists: true,
	}, nil
}

// DefaultWalletPath for the user, which is dependent on operating system.
func (s *Server) DefaultWalletPath(ctx context.Context, _ *ptypes.Empty) (*pb.DefaultWalletResponse, error) {
	return &pb.DefaultWalletResponse{
		WalletDir: filepath.Join(flags.DefaultValidatorDir(), flags.WalletDefaultDirName),
	}, nil
}

// CreateWallet via an API request, allowing a user to save a new
// derived, imported, or remote wallet.
func (s *Server) CreateWallet(ctx context.Context, req *pb.CreateWalletRequest) (*pb.CreateWalletResponse, error) {
	walletDir := s.walletDir
	if strings.TrimSpace(req.WalletPath) != "" {
		walletDir = req.WalletPath
	}
	exists, err := wallet.Exists(walletDir)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not check for existing wallet: %v", err)
	}
	if exists {
		if err := s.initializeWallet(ctx, &wallet.Config{
			WalletDir:      walletDir,
			WalletPassword: req.WalletPassword,
		}); err != nil {
			return nil, err
		}
		keymanagerKind := pb.KeymanagerKind_IMPORTED
		switch s.wallet.KeymanagerKind() {
		case keymanager.Derived:
			keymanagerKind = pb.KeymanagerKind_DERIVED
		case keymanager.Remote:
			keymanagerKind = pb.KeymanagerKind_REMOTE
		}
		return &pb.CreateWalletResponse{
			Wallet: &pb.WalletResponse{
				WalletPath:     walletDir,
				KeymanagerKind: keymanagerKind,
			},
		}, nil
	}
	switch req.Keymanager {
	case pb.KeymanagerKind_IMPORTED:
		w, err := accounts.CreateWalletWithKeymanager(ctx, &accounts.CreateWalletConfig{
			WalletCfg: &wallet.Config{
				WalletDir:      walletDir,
				KeymanagerKind: keymanager.Imported,
				WalletPassword: req.WalletPassword,
			},
			SkipMnemonicConfirm: true,
		})
		if err != nil {
			return nil, err
		}
		if err := w.SaveHashedPassword(ctx); err != nil {
			return nil, err
		}
		if err := s.initializeWallet(ctx, &wallet.Config{
			WalletDir:      walletDir,
			KeymanagerKind: keymanager.Imported,
			WalletPassword: req.WalletPassword,
		}); err != nil {
			return nil, err
		}
		return &pb.CreateWalletResponse{
			Wallet: &pb.WalletResponse{
				WalletPath:     walletDir,
				KeymanagerKind: pb.KeymanagerKind_IMPORTED,
			},
		}, nil
	case pb.KeymanagerKind_DERIVED:
		if req.NumAccounts < 1 {
			return nil, status.Error(codes.InvalidArgument, "Must create at least 1 validator account")
		}
		if req.Mnemonic == "" {
			return nil, status.Error(codes.InvalidArgument, "Must include mnemonic in request")
		}
		_, depositData, err := accounts.RecoverWallet(ctx, &accounts.RecoverWalletConfig{
			WalletDir:      walletDir,
			WalletPassword: req.WalletPassword,
			Mnemonic:       req.Mnemonic,
			NumAccounts:    int64(req.NumAccounts),
		})
		if err != nil {
			return nil, err
		}
		if err := s.initializeWallet(ctx, &wallet.Config{
			WalletDir:      walletDir,
			KeymanagerKind: keymanager.Imported,
			WalletPassword: req.WalletPassword,
		}); err != nil {
			return nil, err
		}

		depositDataList := make([]*pb.DepositDataResponse_DepositData, len(depositData))
		for i, item := range depositData {
			data, err := accounts.DepositDataJSON(item)
			if err != nil {
				return nil, err
			}
			depositDataList[i] = &pb.DepositDataResponse_DepositData{
				Data: data,
			}
		}
		return &pb.CreateWalletResponse{
			Wallet: &pb.WalletResponse{
				WalletPath:     walletDir,
				KeymanagerKind: pb.KeymanagerKind_DERIVED,
			},
			AccountsCreated: &pb.DepositDataResponse{
				DepositDataList: depositDataList,
			},
		}, nil
	case pb.KeymanagerKind_REMOTE:
		return nil, status.Error(codes.Unimplemented, "Remote keymanager not yet supported")
	default:
		return nil, status.Errorf(codes.InvalidArgument, "Keymanager type %T not yet supported", req.Keymanager)
	}
}

// EditConfig allows the user to edit their wallet's keymanageropts.
func (s *Server) EditConfig(_ context.Context, _ *pb.EditWalletConfigRequest) (*pb.WalletResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

// WalletConfig returns the wallet's configuration. If no wallet exists, we return an empty response.
func (s *Server) WalletConfig(ctx context.Context, _ *ptypes.Empty) (*pb.WalletResponse, error) {
	exists, err := wallet.Exists(s.walletDir)
	if err != nil {
		return nil, status.Errorf(codes.Internal, checkExistsErrMsg)
	}
	if !exists {
		// If no wallet is found, we simply return an empty response.
		return &pb.WalletResponse{}, nil
	}
	valid, err := wallet.IsValid(s.walletDir)
	if errors.Is(err, wallet.ErrNoWalletFound) {
		return &pb.WalletResponse{}, nil
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, checkValidityErrMsg)
	}
	if !valid {
		return nil, status.Errorf(codes.FailedPrecondition, invalidWalletMsg)
	}

	if s.wallet == nil || s.keymanager == nil {
		// If no wallet is found, we simply return an empty response.
		return &pb.WalletResponse{}, nil
	}
	var keymanagerKind pb.KeymanagerKind
	switch s.wallet.KeymanagerKind() {
	case keymanager.Derived:
		keymanagerKind = pb.KeymanagerKind_DERIVED
	case keymanager.Imported:
		keymanagerKind = pb.KeymanagerKind_IMPORTED
	case keymanager.Remote:
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
		WalletPath:       s.walletDir,
		KeymanagerKind:   keymanagerKind,
		KeymanagerConfig: config,
	}, nil
}

// GenerateMnemonic creates a new, random bip39 mnemonic phrase.
func (s *Server) GenerateMnemonic(_ context.Context, _ *ptypes.Empty) (*pb.GenerateMnemonicResponse, error) {
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
	exists, err := wallet.Exists(s.walletDir)
	if err != nil {
		return nil, status.Errorf(codes.Internal, checkExistsErrMsg)
	}
	if !exists {
		return nil, status.Errorf(codes.FailedPrecondition, noWalletMsg)
	}
	valid, err := wallet.IsValid(s.walletDir)
	if errors.Is(err, wallet.ErrNoWalletFound) {
		return nil, status.Errorf(codes.FailedPrecondition, noWalletMsg)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, checkValidityErrMsg)
	}
	if !valid {
		return nil, status.Errorf(codes.FailedPrecondition, invalidWalletMsg)
	}
	if req.CurrentPassword == "" {
		return nil, status.Error(codes.InvalidArgument, "Current wallet password cannot be empty")
	}
	hashedPasswordPath := filepath.Join(s.walletDir, wallet.HashedPasswordFileName)
	if !fileutil.FileExists(hashedPasswordPath) {
		return nil, status.Error(codes.FailedPrecondition, "Could not compare password from disk")
	}
	hashedPassword, err := fileutil.ReadFileAsBytes(hashedPasswordPath)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, "Could not retrieve hashed password from disk")
	}
	if err := bcrypt.CompareHashAndPassword(hashedPassword, []byte(req.CurrentPassword)); err != nil {
		return nil, status.Error(codes.Unauthenticated, "Incorrect wallet password")
	}
	if req.Password != req.PasswordConfirmation {
		return nil, status.Error(codes.InvalidArgument, "Password does not match confirmation")
	}
	if err := promptutil.ValidatePasswordInput(req.Password); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Could not validate wallet password input")
	}
	switch s.wallet.KeymanagerKind() {
	case keymanager.Imported:
		km, ok := s.keymanager.(*imported.Keymanager)
		if !ok {
			return nil, status.Error(codes.FailedPrecondition, "Not a valid imported keymanager")
		}
		s.wallet.SetPassword(req.Password)
		if err := s.wallet.SaveHashedPassword(ctx); err != nil {
			return nil, status.Errorf(codes.Internal, "Could not save hashed password: %v", err)
		}
		if err := km.RefreshWalletPassword(ctx); err != nil {
			return nil, status.Errorf(codes.Internal, "Could not refresh wallet password: %v", err)
		}
	case keymanager.Derived:
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
	case keymanager.Remote:
		return nil, status.Error(codes.Internal, "Cannot change password for remote keymanager")
	}
	return &ptypes.Empty{}, nil
}

// ImportKeystores allows importing new keystores via RPC into the wallet
// which will be decrypted using the specified password .
func (s *Server) ImportKeystores(
	ctx context.Context, req *pb.ImportKeystoresRequest,
) (*pb.ImportKeystoresResponse, error) {
	if s.wallet == nil {
		return nil, status.Error(codes.FailedPrecondition, "No wallet initialized")
	}
	km, ok := s.keymanager.(*imported.Keymanager)
	if !ok {
		return nil, status.Error(codes.FailedPrecondition, "Only imported wallets can import more keystores")
	}
	if req.KeystoresPassword == "" {
		return nil, status.Error(codes.InvalidArgument, "Password required for keystores")
	}
	// Needs to unmarshal the keystores from the requests.
	if req.KeystoresImported == nil || len(req.KeystoresImported) < 1 {
		return nil, status.Error(codes.InvalidArgument, "No keystores included for import")
	}
	keystores := make([]*keymanager.Keystore, len(req.KeystoresImported))
	importedPubKeys := make([][]byte, len(req.KeystoresImported))
	for i := 0; i < len(req.KeystoresImported); i++ {
		encoded := req.KeystoresImported[i]
		keystore := &keymanager.Keystore{}
		if err := json.Unmarshal([]byte(encoded), &keystore); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "Not a valid EIP-2335 keystore JSON file: %v", err)
		}
		keystores[i] = keystore
		pubKey, err := hex.DecodeString(keystore.Pubkey)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "Not a valid BLS public key in keystore file: %v", err)
		}
		importedPubKeys[i] = pubKey
	}
	// Import the uploaded accounts.
	if err := accounts.ImportAccounts(ctx, &accounts.ImportAccountsConfig{
		Keymanager:      km,
		Keystores:       keystores,
		AccountPassword: req.KeystoresPassword,
	}); err != nil {
		return nil, err
	}
	s.walletInitializedFeed.Send(s.wallet)
	return &pb.ImportKeystoresResponse{
		ImportedPublicKeys: importedPubKeys,
	}, nil
}
