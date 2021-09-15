package rpc

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/config/features"
	pb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/prysmaticlabs/prysm/shared/promptutil"
	"github.com/prysmaticlabs/prysm/shared/rand"
	"github.com/prysmaticlabs/prysm/validator/accounts"
	"github.com/prysmaticlabs/prysm/validator/accounts/iface"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/imported"
	"github.com/tyler-smith/go-bip39"
	"github.com/tyler-smith/go-bip39/wordlists"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	checkExistsErrMsg   = "Could not check if wallet exists"
	checkValidityErrMsg = "Could not check if wallet is valid"
	invalidWalletMsg    = "Directory does not contain a valid wallet"
)

// CreateWallet via an API request, allowing a user to save a new
// imported wallet via RPC.
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
	if req.Keymanager == pb.KeymanagerKind_IMPORTED {
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
	}
	return nil, status.Errorf(codes.InvalidArgument, "Keymanager type %T create wallet not supported through web", req.Keymanager)
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

// RecoverWallet via an API request, allowing a user to recover a derived.
// Generate the seed from the mnemonic + language + 25th passphrase(optional).
// Create N validator keystores from the seed specified by req.NumAccounts.
// Set the wallet password to req.WalletPassword, then create the wallet from
// the provided Mnemonic and return CreateWalletResponse.
func (s *Server) RecoverWallet(ctx context.Context, req *pb.RecoverWalletRequest) (*pb.CreateWalletResponse, error) {
	numAccounts := int(req.NumAccounts)
	if numAccounts == 0 {
		return nil, status.Error(codes.InvalidArgument, "Must create at least 1 validator account")
	}

	// Check validate mnemonic with chosen language
	language := strings.ToLower(req.Language)
	allowedLanguages := map[string][]string{
		"chinese_simplified":  wordlists.ChineseSimplified,
		"chinese_traditional": wordlists.ChineseTraditional,
		"czech":               wordlists.Czech,
		"english":             wordlists.English,
		"french":              wordlists.French,
		"japanese":            wordlists.Japanese,
		"korean":              wordlists.Korean,
		"italian":             wordlists.Italian,
		"spanish":             wordlists.Spanish,
	}
	if _, ok := allowedLanguages[language]; !ok {
		return nil, status.Error(codes.InvalidArgument, "input not in the list of supported languages")
	}
	bip39.SetWordList(allowedLanguages[language])
	mnemonic := req.Mnemonic
	if err := accounts.ValidateMnemonic(mnemonic); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid mnemonic in request")
	}

	// Check it is not null and not an empty string.
	if req.Mnemonic25ThWord != "" && strings.TrimSpace(req.Mnemonic25ThWord) == "" {
		return nil, status.Error(codes.InvalidArgument, "mnemonic 25th word cannot be empty")
	}

	// Web UI is structured to only write to the default wallet directory
	// accounts.Recoverwallet checks if wallet already exists.
	walletDir := s.walletDir

	// Web UI should check the new and confirmed password are equal.
	walletPassword := req.WalletPassword
	if err := promptutil.ValidatePasswordInput(walletPassword); err != nil {
		return nil, status.Error(codes.InvalidArgument, "password did not pass validation")
	}

	if _, err := accounts.RecoverWallet(ctx, &accounts.RecoverWalletConfig{
		WalletDir:        walletDir,
		WalletPassword:   walletPassword,
		Mnemonic:         mnemonic,
		NumAccounts:      numAccounts,
		Mnemonic25thWord: req.Mnemonic25ThWord,
	}); err != nil {
		return nil, err
	}
	if err := s.initializeWallet(ctx, &wallet.Config{
		WalletDir:      walletDir,
		KeymanagerKind: keymanager.Derived,
		WalletPassword: walletPassword,
	}); err != nil {
		return nil, err
	}
	if err := writeWalletPasswordToDisk(walletDir, walletPassword); err != nil {
		return nil, status.Error(codes.Internal, "Could not write wallet password to disk")
	}
	return &pb.CreateWalletResponse{
		Wallet: &pb.WalletResponse{
			WalletPath:     walletDir,
			KeymanagerKind: pb.KeymanagerKind_DERIVED,
		},
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
	if !features.Get().WriteWalletPasswordOnWebOnboarding {
		return nil
	}
	passwordFilePath := filepath.Join(walletDir, wallet.DefaultWalletPasswordFile)
	if fileutil.FileExists(passwordFilePath) {
		return fmt.Errorf("cannot write wallet password file as it already exists %s", passwordFilePath)
	}
	return fileutil.WriteFile(passwordFilePath, []byte(password))
}
