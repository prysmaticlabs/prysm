package rpc

import (
	"context"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	tokenExpiryLength = 20 * time.Minute
	hashCost          = 8
)

type Claims struct {
	jwt.StandardClaims
}

// Signup --
func (s *Server) Signup(ctx context.Context, req *pb.AuthRequest) (*pb.AuthResponse, error) {
	// Salt and hash the password using the bcrypt algorithm
	// The second argument is the cost of hashing, which we arbitrarily set as 8
	// (this value can be more or less, depending on the computing power you wish to utilize)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), hashCost)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not generate hashed password")
	}
	// Keep the hashed password in-memory for now.
	s.hashedPassword = hashedPassword
	return s.sendAuthResponse()
}

// Login --
func (s *Server) Login(ctx context.Context, req *pb.AuthRequest) (*pb.AuthResponse, error) {
	// Compare the stored hashed password, with the hashed version of the password that was received.
	if err := bcrypt.CompareHashAndPassword(s.hashedPassword, []byte(req.Password)); err != nil {
		return nil, status.Error(codes.Unauthenticated, "Incorrect password")
	}
	return s.sendAuthResponse()
}

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

func (s *Server) createTokenString() (string, uint64, error) {
	expirationTime := time.Now().Add(tokenExpiryLength)
	claims := &Claims{
		StandardClaims: jwt.StandardClaims{
			// In JWT, the expiry time is expressed as unix milliseconds.
			ExpiresAt: expirationTime.Unix(),
		},
	}
	// Declare the token with the algorithm used for signing, and the claims.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtKey)
	if err != nil {
		return "", 0, errors.Wrap(err, "could not sign token")
	}
	return tokenString, uint64(claims.StandardClaims.ExpiresAt), nil
}
