package rpc

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/prysmaticlabs/prysm/io/file"
	pb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	prysmTime "github.com/prysmaticlabs/prysm/time"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	tokenExpiryLength = time.Hour
)

const (
	authTokenFileName = "auth-token"
)

// Initialize returns metadata regarding whether the caller has authenticated and has a wallet.
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
func (s *Server) saveAuthToken(token string, expiration uint64) error {
	hashFilePath := filepath.Join(s.walletDir, authTokenFileName)
	bytesBuf := new(bytes.Buffer)
	bytesBuf.Write([]byte(token))
	bytesBuf.Write([]byte("\n"))
	bytesBuf.Write([]byte(fmt.Sprintf("%d", expiration)))
	return file.WriteFile(hashFilePath, bytesBuf.Bytes())
}

// Upon launch of the validator client, we initialize an auth token by either creating
// one from scratch or reading it from a file. This token can then be shown to the
// user via stdout and the validator client should then attempt to open the default
// browser. The web interface authenticates by looking for this token in the query parameters
// of the URL. This token is then used as the bearer token for jwt auth.
func (s *Server) initializeAuthToken() (string, uint64, error) {
	authTokenFile := filepath.Join(s.walletDir, authTokenFileName)
	if file.FileExists(authTokenFile) {
		f, err := os.Open(authTokenFile)
		if err != nil {
			return "", 0, err
		}
		r := bufio.NewReader(f)
		token, err := r.ReadString('\n')
		if err != nil {
			return "", 0, err
		}
		exprBytes, _, err := r.ReadLine()
		if err != nil {
			return "", 0, err
		}
		expiration, err := strconv.ParseUint(string(exprBytes), 10, 8)
		if err != nil {
			return "", 0, err
		}
		return token, expiration, nil
	}
	token, expiration, err := s.createTokenString()
	if err != nil {
		return "", 0, err
	}
	return token, expiration, nil
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
