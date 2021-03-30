package rpc

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/prysmaticlabs/prysm/shared/rand"
	"github.com/prysmaticlabs/prysm/validator/accounts"
	"github.com/prysmaticlabs/prysm/validator/accounts/iface"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/imported"
	"github.com/tyler-smith/go-bip39"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	checkExistsErrMsg   = "Could not check if wallet exists"
	checkValidityErrMsg = "Could not check if wallet is valid"
	invalidWalletMsg    = "Directory does not contain a valid wallet"
)

// CreateWallet via an API request, allowing a user to save a new
// derived, imported, or remote wallet.
func (s *Server) CreateWallet(ctx context.Context, req *pb.CreateWalletRequest) (*pb.CreateWalletResponse, error) {
	walletDir := s.walletDir
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
		_, err := accounts.CreateWalletWithKeymanager(ctx, &accounts.CreateWalletConfig{
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
		if err := s.initializeWallet(ctx, &wallet.Config{
			WalletDir:      walletDir,
			KeymanagerKind: keymanager.Imported,
			WalletPassword: req.WalletPassword,
		}); err != nil {
			return nil, err
		}
		if err := writeWalletPasswordToDisk(walletDir, req.WalletPassword); err != nil {
			return nil, status.Error(codes.Internal, "Could not write wallet password to disk")
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
		if _, err := accounts.RecoverWallet(ctx, &accounts.RecoverWalletConfig{
			WalletDir:      walletDir,
			WalletPassword: req.WalletPassword,
			Mnemonic:       req.Mnemonic,
			NumAccounts:    int(req.NumAccounts),
		}); err != nil {
			return nil, err
		}
		if err := s.initializeWallet(ctx, &wallet.Config{
			WalletDir:      walletDir,
			KeymanagerKind: keymanager.Imported,
			WalletPassword: req.WalletPassword,
		}); err != nil {
			return nil, err
		}
		if err := writeWalletPasswordToDisk(walletDir, req.WalletPassword); err != nil {
			return nil, status.Error(codes.Internal, "Could not write wallet password to disk")
		}
		return &pb.CreateWalletResponse{
			Wallet: &pb.WalletResponse{
				WalletPath:     walletDir,
				KeymanagerKind: pb.KeymanagerKind_DERIVED,
			},
		}, nil
	case pb.KeymanagerKind_REMOTE:
		return nil, status.Error(codes.Unimplemented, "Remote keymanager not yet supported")
	default:
		return nil, status.Errorf(codes.InvalidArgument, "Keymanager type %T not yet supported", req.Keymanager)
	}
}

// WalletConfig returns the wallet's configuration. If no wallet exists, we return an empty response.
func (s *Server) WalletConfig(ctx context.Context, _ *empty.Empty) (*pb.WalletResponse, error) {
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
	return &pb.WalletResponse{
		WalletPath:     s.walletDir,
		KeymanagerKind: keymanagerKind,
	}, nil
}

// GenerateMnemonic creates a new, random bip39 mnemonic phrase.
func (s *Server) GenerateMnemonic(_ context.Context, _ *empty.Empty) (*pb.GenerateMnemonicResponse, error) {
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

// Initialize a wallet and send it over a global feed.
func (s *Server) initializeWallet(ctx context.Context, cfg *wallet.Config) error {
	// We first ensure the user has a wallet.
	exists, err := wallet.Exists(cfg.WalletDir)
	if err != nil {
		return errors.Wrap(err, wallet.CheckExistsErrMsg)
	}
	if !exists {
		return wallet.ErrNoWalletFound
	}
	valid, err := wallet.IsValid(cfg.WalletDir)
	if errors.Is(err, wallet.ErrNoWalletFound) {
		return wallet.ErrNoWalletFound
	}
	if err != nil {
		return errors.Wrap(err, wallet.CheckValidityErrMsg)
	}
	if !valid {
		return errors.New(wallet.InvalidWalletErrMsg)
	}

	// We fire an event with the opened wallet over
	// a global feed signifying wallet initialization.
	w, err := wallet.OpenWallet(ctx, &wallet.Config{
		WalletDir:      cfg.WalletDir,
		WalletPassword: cfg.WalletPassword,
	})
	if err != nil {
		return errors.Wrap(err, "could not open wallet")
	}

	s.walletInitialized = true
	km, err := w.InitializeKeymanager(ctx, iface.InitKeymanagerConfig{ListenForChanges: true})
	if err != nil {
		return errors.Wrap(err, accounts.ErrCouldNotInitializeKeymanager)
	}
	s.keymanager = km
	s.wallet = w
	s.walletDir = cfg.WalletDir

	// Only send over feed if we have validating keys.
	validatingPublicKeys, err := km.FetchValidatingPublicKeys(ctx)
	if err != nil {
		return errors.Wrap(err, "could not check for validating public keys")
	}
	if len(validatingPublicKeys) > 0 {
		s.walletInitializedFeed.Send(w)
	}
	return nil
}

func writeWalletPasswordToDisk(walletDir, password string) error {
	if !featureconfig.Get().WriteWalletPasswordOnWebOnboarding {
		return nil
	}
	passwordFilePath := filepath.Join(walletDir, wallet.DefaultWalletPasswordFile)
	if fileutil.FileExists(passwordFilePath) {
		return fmt.Errorf("cannot write wallet password file as it already exists %s", passwordFilePath)
	}
	return fileutil.WriteFile(passwordFilePath, []byte(password))
}
