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
)

var (
	tokenExpiryLength = time.Hour
	hashCost          = 8
)

const (
	// HashedRPCPassword for the validator RPC access.
	HashedRPCPassword       = "rpc-password-hash"
	authToken               = "auth-token"
	checkUserSignupInterval = time.Second * 30
)

// Signup to authenticate access to the validator RPC API using bcrypt and
// a sufficiently strong password check.
//func (s *Server) Signup(ctx context.Context, req *pb.AuthRequest) (*pb.AuthResponse, error) {
//walletDir := s.walletDir
//if req.Password != req.PasswordConfirmation {
//return nil, status.Error(codes.InvalidArgument, "Password confirmation does not match")
//}
//// First, we check if the validator already has a password. In this case,
//// the user should be logged in as normal.
//if file.FileExists(filepath.Join(walletDir, HashedRPCPassword)) {
//return s.Login(ctx, req)
//}
//// We check the strength of the password to ensure it is high-entropy,
//// has the required character count, and contains only unicode characters.
//if err := prompt.ValidatePasswordInput(req.Password); err != nil {
//return nil, status.Errorf(codes.InvalidArgument, "Could not validate RPC password input: %v", err)
//}
//hasDir, err := file.HasDir(walletDir)
//if err != nil {
//return nil, status.Error(codes.FailedPrecondition, "Could not check if wallet directory exists")
//}
//if !hasDir {
//if err := file.MkdirAll(walletDir); err != nil {
//return nil, status.Errorf(codes.Internal, "could not write directory %s to disk: %v", walletDir, err)
//}
//}
//// Write the password hash to disk.
//if err := s.SaveHashedPassword(req.Password); err != nil {
//return nil, status.Errorf(codes.Internal, "could not write hashed password to disk: %v", err)
//}
//return s.sendAuthResponse()
//}

//// Login to authenticate with the validator RPC API using a password.
//func (s *Server) Login(_ context.Context, req *pb.AuthRequest) (*pb.AuthResponse, error) {
//walletDir := s.walletDir
//// We check the strength of the password to ensure it is high-entropy,
//// has the required character count, and contains only unicode characters.
//if err := prompt.ValidatePasswordInput(req.Password); err != nil {
//return nil, status.Errorf(codes.InvalidArgument, "Could not validate RPC password input: %v", err)
//}
//hashedPasswordPath := filepath.Join(walletDir, HashedRPCPassword)
//if !file.FileExists(hashedPasswordPath) {
//return nil, status.Error(codes.Internal, "Could not find hashed password on disk")
//}
//hashedPassword, err := file.ReadFileAsBytes(hashedPasswordPath)
//if err != nil {
//return nil, status.Error(codes.Internal, "Could not retrieve hashed password from disk")
//}
//// Compare the stored hashed password, with the hashed version of the password that was received.
//if err := bcrypt.CompareHashAndPassword(hashedPassword, []byte(req.Password)); err != nil {
//return nil, status.Error(codes.Unauthenticated, "Incorrect validator RPC password")
//}
//return s.sendAuthResponse()
//}

// HasUsedWeb checks if the user has authenticated via the web interface.
func (s *Server) Initialize(_ context.Context, req *pb.InitializeAuthRequest) (*pb.InitializeAuthResponse, error) {
	if file.FileExists(filepath.Join(s.walletDir, authTokenHash)) {
		return nil, nil
	}
	walletExists, err := wallet.Exists(s.walletDir)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not check if wallet exists")
	}
	hashedPasswordPath := filepath.Join(s.walletDir, HashedRPCPassword)
	return &pb.InitializeAuthResponse{
		HasSignedUp: file.FileExists(hashedPasswordPath),
		HasWallet:   walletExists,
	}, nil
}

// Sends an auth response via gRPC containing a new JWT token.
//func (s *Server) sendAuthResponse() (*pb.AuthResponse, error) {
//// If everything is fine here, construct the auth token.
//tokenString, expirationTime, err := s.createTokenString()
//if err != nil {
//return nil, status.Error(codes.Internal, "Could not create jwt token string")
//}
//return &pb.AuthResponse{
//Token:           tokenString,
//TokenExpiration: expirationTime,
//}, nil
//}

// ChangePassword allows changing the RPC password via the API as an authenticated method.
//func (s *Server) ChangePassword(_ context.Context, req *pb.ChangePasswordRequest) (*empty.Empty, error) {
//if req.CurrentPassword == "" {
//return nil, status.Error(codes.InvalidArgument, "Current password cannot be empty")
//}
//hashedPasswordPath := filepath.Join(s.walletDir, HashedRPCPassword)
//if !file.FileExists(hashedPasswordPath) {
//return nil, status.Error(codes.FailedPrecondition, "Could not compare password from disk")
//}
//hashedPassword, err := file.ReadFileAsBytes(hashedPasswordPath)
//if err != nil {
//return nil, status.Error(codes.FailedPrecondition, "Could not retrieve hashed password from disk")
//}
//if err := bcrypt.CompareHashAndPassword(hashedPassword, []byte(req.CurrentPassword)); err != nil {
//return nil, status.Error(codes.Unauthenticated, "Incorrect password")
//}
//if req.Password != req.PasswordConfirmation {
//return nil, status.Error(codes.InvalidArgument, "Password does not match confirmation")
//}
//if err := prompt.ValidatePasswordInput(req.Password); err != nil {
//return nil, status.Errorf(codes.InvalidArgument, "Could not validate password input: %v", err)
//}
//// Write the new password hash to disk.
//if err := s.SaveHashedPassword(req.Password); err != nil {
//return nil, status.Errorf(codes.Internal, "could not write hashed password to disk: %v", err)
//}
//return &empty.Empty{}, nil
//}

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
	authTokenFile := filepath.Join(s.walletDir, authToken)
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
	if file.FileExists(filepath.Join(s.walletDir, authToken)) {
		return nil
	}
}

// Interval in which we should check if a user has not yet used the RPC Signup endpoint
// which means they are using the --web flag and someone could come in and signup for them
// if they have their web host:port exposed to the Internet.
func (s *Server) checkUserSignup(_ context.Context) {
	ticker := time.NewTicker(checkUserSignupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			hashedPasswordPath := filepath.Join(s.walletDir, HashedRPCPassword)
			if file.FileExists(hashedPasswordPath) {
				return
			}
			log.Warnf(
				"You are using the --web option but have not yet signed via a browser. "+
					"If your web host and port are exposed to the Internet, someone else can attempt to sign up "+
					"for you! You can visit http://%s:%d to view the Prysm web interface",
				s.validatorGatewayHost,
				s.validatorGatewayPort,
			)
		case <-s.ctx.Done():
			return
		}
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
