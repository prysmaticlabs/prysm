package rpc

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/promptutil"
	"github.com/prysmaticlabs/prysm/shared/timeutils"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	tokenExpiryLength = time.Hour
	hashCost          = 8
)

const (
	// HashedRPCPassword for the validator RPC access.
	HashedRPCPassword = "rpc-password-hash"
)

// Signup to authenticate access to the validator RPC API using bcrypt and
// a sufficiently strong password check.
func (s *Server) Signup(ctx context.Context, req *pb.AuthRequest) (*pb.AuthResponse, error) {
	walletDir := s.walletDir
	if strings.TrimSpace(req.WalletDir) != "" {
		walletDir = req.WalletDir
	}
	// First, we check if the validator already has a password. In this case,
	// the user should be logged in as normal.
	if fileutil.FileExists(filepath.Join(walletDir, HashedRPCPassword)) {
		return s.Login(ctx, req)
	}
	// We check the strength of the password to ensure it is high-entropy,
	// has the required character count, and contains only unicode characters.
	if err := promptutil.ValidatePasswordInput(req.Password); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Could not validate RPC password input")
	}
	hasDir, err := fileutil.HasDir(walletDir)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, "Could not check if wallet directory exists")
	}
	if !hasDir {
		if err := os.MkdirAll(walletDir, os.ModePerm); err != nil {
			return nil, status.Errorf(codes.Internal, "could not write directory %s to disk: %v", walletDir, err)
		}
	}
	// Write the password hash to disk.
	if err := s.SaveHashedPassword(req.Password); err != nil {
		return nil, status.Errorf(codes.Internal, "could not write hashed password to disk: %v", err)
	}
	return s.sendAuthResponse()
}

// Login to authenticate with the validator RPC API using a password.
func (s *Server) Login(ctx context.Context, req *pb.AuthRequest) (*pb.AuthResponse, error) {
	walletDir := s.walletDir
	if strings.TrimSpace(req.WalletDir) != "" {
		walletDir = req.WalletDir
	}
	// We check the strength of the password to ensure it is high-entropy,
	// has the required character count, and contains only unicode characters.
	if err := promptutil.ValidatePasswordInput(req.Password); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Could not validate RPC password input")
	}
	hashedPasswordPath := filepath.Join(walletDir, HashedRPCPassword)
	if !fileutil.FileExists(hashedPasswordPath) {
		return nil, status.Error(codes.Internal, "Could not find hashed password on disk")
	}
	hashedPassword, err := fileutil.ReadFileAsBytes(hashedPasswordPath)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not retrieve hashed password from disk")
	}
	// Compare the stored hashed password, with the hashed version of the password that was received.
	if err := bcrypt.CompareHashAndPassword(hashedPassword, []byte(req.Password)); err != nil {
		return nil, status.Error(codes.Unauthenticated, "Incorrect validator RPC password")
	}
	return s.sendAuthResponse()
}

// Logout a user by invalidating their JWT key.
func (s *Server) Logout(ctx context.Context, _ *ptypes.Empty) (*ptypes.Empty, error) {
	// Invalidate the old JWT key, making all requests done with its token fail.
	jwtKey, err := createRandomJWTKey()
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not invalidate JWT key")
	}
	s.jwtKey = jwtKey
	return &ptypes.Empty{}, nil
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

// ChangePassword allows changing the RPC password via the API as an authenticated method.
func (s *Server) ChangePassword(ctx context.Context, req *pb.ChangePasswordRequest) (*ptypes.Empty, error) {
	if req.CurrentPassword == "" {
		return nil, status.Error(codes.InvalidArgument, "Current password cannot be empty")
	}
	hashedPasswordPath := filepath.Join(s.walletDir, HashedRPCPassword)
	if !fileutil.FileExists(hashedPasswordPath) {
		return nil, status.Error(codes.FailedPrecondition, "Could not compare password from disk")
	}
	hashedPassword, err := fileutil.ReadFileAsBytes(hashedPasswordPath)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, "Could not retrieve hashed password from disk")
	}
	if err := bcrypt.CompareHashAndPassword(hashedPassword, []byte(req.CurrentPassword)); err != nil {
		return nil, status.Error(codes.Unauthenticated, "Incorrect password")
	}
	if req.Password != req.PasswordConfirmation {
		return nil, status.Error(codes.InvalidArgument, "Password does not match confirmation")
	}
	if err := promptutil.ValidatePasswordInput(req.Password); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Could not validate password input")
	}
	// Write the new password hash to disk.
	if err := s.SaveHashedPassword(req.Password); err != nil {
		return nil, status.Errorf(codes.Internal, "could not write hashed password to disk: %v", err)
	}
	return &ptypes.Empty{}, nil
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

// SaveHashedPassword to disk for the validator RPC.
func (s *Server) SaveHashedPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), hashCost)
	if err != nil {
		return errors.Wrap(err, "could not generate hashed password")
	}
	hashFilePath := filepath.Join(s.walletDir, HashedRPCPassword)
	return ioutil.WriteFile(hashFilePath, hashedPassword, params.BeaconIoConfig().ReadWritePermissions)
}
