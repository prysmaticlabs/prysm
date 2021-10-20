package rpc

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/time"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestServer_JWTInterceptor_Verify(t *testing.T) {
	s := Server{
		jwtKey: []byte("testKey"),
	}
	interceptor := s.JWTInterceptor()

	unaryInfo := &grpc.UnaryServerInfo{
		FullMethod: "Proto.CreateWallet",
	}
	unaryHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}
	token, _, err := createTokenString(s.jwtKey)
	require.NoError(t, err)
	ctxMD := map[string][]string{
		"authorization": {"Bearer " + token},
	}
	ctx := context.Background()
	ctx = metadata.NewIncomingContext(ctx, ctxMD)
	_, err = interceptor(ctx, "xyz", unaryInfo, unaryHandler)
	require.NoError(t, err)
}

func TestServer_JWTInterceptor_BadToken(t *testing.T) {
	s := Server{
		jwtKey: []byte("testKey"),
	}
	interceptor := s.JWTInterceptor()

	unaryInfo := &grpc.UnaryServerInfo{
		FullMethod: "Proto.CreateWallet",
	}
	unaryHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	badServer := Server{
		jwtKey: []byte("badTestKey"),
	}
	token, _, err := createTokenString(badServer.jwtKey)
	require.NoError(t, err)
	ctxMD := map[string][]string{
		"authorization": {"Bearer " + token},
	}
	ctx := context.Background()
	ctx = metadata.NewIncomingContext(ctx, ctxMD)
	_, err = interceptor(ctx, "xyz", unaryInfo, unaryHandler)
	require.ErrorContains(t, "signature is invalid", err)
}

func TestServer_JWTInterceptor_InvalidSigningType(t *testing.T) {
	ss := &Server{jwtKey: make([]byte, 32)}
	expirationTime := time.Now().Add(tokenExpiryLength)
	// Use a different signing type than the expected, HMAC.
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.StandardClaims{
		ExpiresAt: expirationTime.Unix(),
	})
	_, err := ss.validateJWT(token)
	require.ErrorContains(t, "unexpected JWT signing method", err)
}
