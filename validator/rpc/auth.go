package rpc

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/promptutil"
	"github.com/prysmaticlabs/prysm/shared/timeutils"
	"github.com/prysmaticlabs/prysm/validator/accounts/v2/wallet"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	tokenExpiryLength = 20 * time.Minute
	hashCost          = 8
)

// Signup to authenticate access to the validator RPC API using bcrypt and
// a sufficiently strong password check.
func (s *Server) Signup(ctx context.Context, req *pb.AuthRequest) (*pb.AuthResponse, error) {
	// First, we check if the validator already has a password. In this case,
	// the user should NOT be able to signup and the function will return an error.
	if fileutil.FileExists(filepath.Join(defaultWalletPath, wallet.HashedPasswordFileName)) {
		return nil, status.Error(codes.PermissionDenied, "Validator already has a password set, cannot signup")
	}
	// We check the strength of the password to ensure it is high-entropy,
	// has the required character count, and contains only unicode characters.
	if err := promptutil.ValidatePasswordInput(req.Password); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Could not validate password input")
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), hashCost)
	if err != nil {
		return nil, errors.Wrap(err, "could not generate hashed password")
	}
	hashFilePath := filepath.Join(defaultWalletPath, wallet.HashedPasswordFileName)
	// Write the config file to disk.
	if err := os.MkdirAll(defaultWalletPath, os.ModePerm); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if err := ioutil.WriteFile(hashFilePath, hashedPassword, params.BeaconIoConfig().ReadWritePermissions); err != nil {
		return nil, status.Errorf(codes.Internal, "could not write hashed password for wallet to disk: %v", err)
	}
	return s.sendAuthResponse()
}

// Login to authenticate with the validator RPC API using a password.
func (s *Server) Login(ctx context.Context, req *pb.AuthRequest) (*pb.AuthResponse, error) {
	hashedPasswordPath := filepath.Join(defaultWalletPath, wallet.HashedPasswordFileName)
	if fileutil.FileExists(hashedPasswordPath) {
		hashedPassword, err := fileutil.ReadFileAsBytes(hashedPasswordPath)
		if err != nil {
			return nil, status.Error(codes.Internal, "Could not retrieve hashed password from disk")
		}
		// Compare the stored hashed password, with the hashed version of the password that was received.
		if err := bcrypt.CompareHashAndPassword(hashedPassword, []byte(req.Password)); err != nil {
			return nil, status.Error(codes.Unauthenticated, "Incorrect password")
		}
	}
	if err := s.initializeWallet(ctx, &wallet.Config{
		WalletDir:      defaultWalletPath,
		WalletPassword: req.Password,
	}); err != nil {
		if strings.Contains(err.Error(), "invalid checksum") {
			return nil, status.Error(codes.Unauthenticated, "Incorrect password")
		}
		return nil, status.Errorf(codes.Internal, "Could not initialize wallet: %v", err)
	}
	return s.sendAuthResponse()
}

// Sends an auth response via gRPC containing a new JWT token.
func (s *Server) sendAuthResponse() (*pb.AuthResponse, error) {
	// If everything is fine here, construct the auth token.
	tokenString, expirationTime, err := s.createTokenString()
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not create jwt token string")
	}
	return &pb.AuthResponse{
		Token:           tokenString,
		TokenExpiration: expirationTime,
	}, nil
}

// Creates a JWT token string using the JWT key with an expiration timestamp.
func (s *Server) createTokenString() (string, uint64, error) {
	// Create a new token object, specifying signing method and the claims
	// you would like it to contain.
	expirationTime := timeutils.Now().Add(tokenExpiryLength)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		ExpiresAt: expirationTime.Unix(),
	})
	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString(s.jwtKey)
	if err != nil {
		return "", 0, err
	}
	return tokenString, uint64(expirationTime.Unix()), nil
}

// Initialize a wallet and send it over a global feed.
func (s *Server) initializeWallet(ctx context.Context, cfg *wallet.Config) error {
	// We first ensure the user has a wallet.
	if err := wallet.Exists(cfg.WalletDir); err != nil {
		if errors.Is(err, wallet.ErrNoWalletFound) {
			return wallet.ErrNoWalletFound
		}
		return errors.Wrap(err, "could not check if wallet exists")
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
	km, err := w.InitializeKeymanager(ctx, true /* skip mnemonic confirm */)
	if err != nil {
		return errors.Wrap(err, "could not initialize keymanager")
	}
	s.keymanager = km
	s.wallet = w
	s.walletInitializedFeed.Send(w)
	return nil
}
