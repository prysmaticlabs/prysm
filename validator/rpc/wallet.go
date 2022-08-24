package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/config/features"
	"github.com/prysmaticlabs/prysm/v3/io/file"
	"github.com/prysmaticlabs/prysm/v3/io/prompt"
	pb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/v3/validator/accounts"
	"github.com/prysmaticlabs/prysm/v3/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager"
	"github.com/tyler-smith/go-bip39"
	"github.com/tyler-smith/go-bip39/wordlists"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
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
		case keymanager.Web3Signer:
			keymanagerKind = pb.KeymanagerKind_WEB3SIGNER
		}
		return &pb.CreateWalletResponse{
			Wallet: &pb.WalletResponse{
				WalletPath:     walletDir,
				KeymanagerKind: keymanagerKind,
			},
		}, nil
	}
	if err := prompt.ValidatePasswordInput(req.WalletPassword); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Password too weak: %v", err)
	}
	if req.Keymanager == pb.KeymanagerKind_IMPORTED {
		_, err := accounts.CreateWalletWithKeymanager(ctx, &accounts.CreateWalletConfig{
			WalletCfg: &wallet.Config{
				WalletDir:      walletDir,
				KeymanagerKind: keymanager.Local,
				WalletPassword: req.WalletPassword,
			},
			SkipMnemonicConfirm: true,
		})
		if err != nil {
			return nil, err
		}
		if err := s.initializeWallet(ctx, &wallet.Config{
			WalletDir:      walletDir,
			KeymanagerKind: keymanager.Local,
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
func (s *Server) WalletConfig(_ context.Context, _ *empty.Empty) (*pb.WalletResponse, error) {
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

	if s.wallet == nil || s.validatorService == nil {
		// If no wallet is found, we simply return an empty response.
		return &pb.WalletResponse{}, nil
	}
	var keymanagerKind pb.KeymanagerKind
	switch s.wallet.KeymanagerKind() {
	case keymanager.Derived:
		keymanagerKind = pb.KeymanagerKind_DERIVED
	case keymanager.Local:
		keymanagerKind = pb.KeymanagerKind_IMPORTED
	case keymanager.Remote:
		keymanagerKind = pb.KeymanagerKind_REMOTE
	case keymanager.Web3Signer:
		keymanagerKind = pb.KeymanagerKind_WEB3SIGNER
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
	if err := prompt.ValidatePasswordInput(walletPassword); err != nil {
		return nil, status.Error(codes.InvalidArgument, "password did not pass validation")
	}

	opts := []accounts.Option{
		accounts.WithWalletDir(walletDir),
		accounts.WithWalletPassword(walletPassword),
		accounts.WithMnemonic(mnemonic),
		accounts.WithMnemonic25thWord(req.Mnemonic25ThWord),
		accounts.WithNumAccounts(numAccounts),
	}
	acc, err := accounts.NewCLIManager(opts...)
	if err != nil {
		return nil, err
	}
	if _, err := acc.WalletRecover(ctx); err != nil {
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

// ValidateKeystores checks whether a set of EIP-2335 keystores in the request
// can indeed be decrypted using a password in the request. If there is no issue,
// we return an empty response with no error. If the password is incorrect for a single keystore,
// we return an appropriate error.
func (_ *Server) ValidateKeystores(
	_ context.Context, req *pb.ValidateKeystoresRequest,
) (*emptypb.Empty, error) {
	if req.KeystoresPassword == "" {
		return nil, status.Error(codes.InvalidArgument, "Password required for keystores")
	}
	// Needs to unmarshal the keystores from the requests.
	if req.Keystores == nil || len(req.Keystores) < 1 {
		return nil, status.Error(codes.InvalidArgument, "No keystores included in request")
	}
	decryptor := keystorev4.New()
	for i := 0; i < len(req.Keystores); i++ {
		encoded := req.Keystores[i]
		keystore := &keymanager.Keystore{}
		if err := json.Unmarshal([]byte(encoded), &keystore); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "Not a valid EIP-2335 keystore JSON file: %v", err)
		}
		if _, err := decryptor.Decrypt(keystore.Crypto, req.KeystoresPassword); err != nil {
			doesNotDecrypt := strings.Contains(err.Error(), keymanager.IncorrectPasswordErrMsg)
			if doesNotDecrypt {
				return nil, status.Errorf(
					codes.InvalidArgument,
					"Password for keystore with public key %s is incorrect. "+
						"Prysm web only supports importing batches of keystores with the same password for all of them",
					keystore.Pubkey,
				)
			} else {
				return nil, status.Errorf(codes.Internal, "Unexpected error decrypting keystore: %v", err)
			}
		}
	}

	return &emptypb.Empty{}, nil
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
	s.wallet = w
	s.walletDir = cfg.WalletDir

	s.walletInitializedFeed.Send(w)

	return nil
}

func writeWalletPasswordToDisk(walletDir, password string) error {
	if !features.Get().WriteWalletPasswordOnWebOnboarding {
		return nil
	}
	passwordFilePath := filepath.Join(walletDir, wallet.DefaultWalletPasswordFile)
	if file.FileExists(passwordFilePath) {
		return fmt.Errorf("cannot write wallet password file as it already exists %s", passwordFilePath)
	}
	return file.WriteFile(passwordFilePath, []byte(password))
}
