package rpc

import (
	"context"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/promptutil"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
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
	existingPassword, err := s.valDB.HashedPasswordForAPI(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not retrieve hashed password from database")
	}
	if len(existingPassword) != 0 {
		return nil, status.Error(codes.PermissionDenied, "Validator already has a password set, cannot signup")
	}
	// We check the strength of the password to ensure it is high-entropy,
	// has the required character count, and contains only unicode characters.
	if err := promptutil.ValidatePasswordInput(req.Password); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Could not validate password input")
	}
	// Salt and hash the password using the bcrypt algorithm
	// The second argument is the cost of hashing, which we arbitrarily set as 8
	// (this value can be more or less, depending on the computing power you wish to utilize)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), hashCost)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not generate hashed password")
	}
	// We store the hashed password to disk.
	if err := s.valDB.SaveHashedPasswordForAPI(ctx, hashedPassword); err != nil {
		return nil, status.Error(codes.Internal, "Could not save hashed password to database")
	}
	return s.sendAuthResponse()
}

// Login to authenticate with the validator RPC API using a password.
func (s *Server) Login(ctx context.Context, req *pb.AuthRequest) (*pb.AuthResponse, error) {
	// We retrieve the hashed password for the validator API from disk.
	hashedPassword, err := s.valDB.HashedPasswordForAPI(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not retrieve hashed password from database")
	}
	// Compare the stored hashed password, with the hashed version of the password that was received.
	if err := bcrypt.CompareHashAndPassword(hashedPassword, []byte(req.Password)); err != nil {
		return nil, status.Error(codes.Unauthenticated, "Incorrect password")
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
		Token:           []byte(tokenString),
		TokenExpiration: expirationTime,
	}, nil
}

// Creates a JWT token string using the JWT key with an expiration timestamp.
func (s *Server) createTokenString() (string, uint64, error) {
	expirationTime := roughtime.Now().Add(tokenExpiryLength)
	claims := &jwt.StandardClaims{
		// In JWT, the expiry time is expressed as unix milliseconds.
		ExpiresAt: expirationTime.Unix(),
	}
	// Declare the token with the algorithm used for signing, and the claims.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtKey)
	if err != nil {
		return "", 0, errors.Wrap(err, "could not sign token")
	}
	return tokenString, uint64(claims.ExpiresAt), nil
}
