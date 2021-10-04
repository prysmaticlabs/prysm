package rpc

import (
	"context"
	"path/filepath"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/io/file"
	pb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	prysmTime "github.com/prysmaticlabs/prysm/time"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	tokenExpiryLength = time.Hour
)

const (
	// HashedRPCPassword for the validator RPC access.
	HashedRPCPassword       = "rpc-password-hash"
	authTokenFileName       = "auth-token"
	checkUserSignupInterval = time.Second * 30
)

// HasUsedWeb checks if the user has authenticated via the web interface.
func (s *Server) Initialize(_ context.Context, _ *emptypb.Empty) (*pb.InitializeAuthResponse, error) {
	walletExists, err := wallet.Exists(s.walletDir)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not check if wallet exists")
	}
	authTokenPath := filepath.Join(s.walletDir, authTokenFileName)
	return &pb.InitializeAuthResponse{
		HasSignedUp: file.FileExists(authTokenPath),
		HasWallet:   walletExists,
	}, nil
}

// SaveHashedPassword to disk for the validator RPC.
func (s *Server) saveHashedAuthToken(token string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(token), hashCost)
	if err != nil {
		return errors.Wrap(err, "could not generate hashed password")
	}
	hashFilePath := filepath.Join(s.walletDir, HashedRPCPassword)
	return file.WriteFile(hashFilePath, hashedPassword)
}

// Upon launch of the validator client, we initialize an auth token by either creating
// one from scratch or reading it from a file. This token can then be shown to the
// user via stdout and the validator client should then attempt to open the default
// browser. The web interface authenticates by looking for this token in the query parameters
// of the URL. This token is then used as the bearer token for jwt auth.
func (s *Server) initializeAuthToken() error {
	authTokenFile := filepath.Join(s.walletDir, authTokenFileName)
	if file.FileExists(authTokenFile) {
		authToken, err := file.ReadFileAsBytes(authTokenFile)
		if err != nil {
			return err
		}
		return nil
	}
	token, expr, err := s.createTokenString()
	if err != nil {
		return err
	}
	if file.FileExists(filepath.Join(s.walletDir, authTokenFileName)) {
		return nil
	}
}

// Creates a JWT token string using the JWT key with an expiration timestamp.
func (s *Server) createTokenString() (string, uint64, error) {
	// Create a new token object, specifying signing method and the claims
	// you would like it to contain.
	expirationTime := prysmTime.Now().Add(tokenExpiryLength)
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
