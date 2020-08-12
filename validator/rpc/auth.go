package rpc

import (
	"context"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

var (
	tokenExpiryLength = 20 * time.Minute
	hashCost          = 8
)

type Claims struct {
	jwt.StandardClaims
}

// Signup --
func (s *Server) Signup(ctx context.Context, req *pb.SignupRequest) (*pb.SignupResponse, error) {
	// Salt and hash the password using the bcrypt algorithm
	// The second argument is the cost of hashing, which we arbitrarily set as 8
	// (this value can be more or less, depending on the computing power you wish to utilize)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), hashCost)

	// Keep the hashed password in-memory for now.
	s.hashedPassword = hashedPassword

	tokenString, err := s.createTokenString()
	if err != nil {
		return nil, errors.Wrap(err, "could not get token for user")
	}
	resp := &pb.SignupResponse{
		Token: []byte(tokenString),
	}
	return resp, nil
}

// Login --
func (s *Server) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	// Compare the stored hashed password, with the hashed version of the password that was received.
	if err = bcrypt.CompareHashAndPassword(s.hashedPassword, []byte(req.Password)); err != nil {
		return nil, errors.New("incorrect password")
	}

	// If everything is fine here, construct the auth token.
	tokenString, err := s.createTokenString()
	if err != nil {
		return nil, errors.Wrap(err, "could not get token for user")
	}
	resp := &pb.SignupResponse{
		Token: []byte(tokenString),
	}
	return resp, nil
}

func (s *Server) createTokenString() (string, error) {
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
		return "", errors.Wrap(err, "could not sign token")
	}
	return tokenString, nil
}
