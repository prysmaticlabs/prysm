package rpc

import (
	"context"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
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
	token, _, err := s.createTokenString()
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
	token, _, err := badServer.createTokenString()
	require.NoError(t, err)
	ctxMD := map[string][]string{
		"authorization": {"Bearer " + token},
	}
	ctx := context.Background()
	ctx = metadata.NewIncomingContext(ctx, ctxMD)
	_, err = interceptor(ctx, "xyz", unaryInfo, unaryHandler)
	if err == nil {
		t.Fatalf("Unexpected success processing token %v", err)
	}
	if !strings.Contains(err.Error(), "signature is invalid") {
		t.Fatalf("Expected error validating signature, received %v", err)
	}
}
