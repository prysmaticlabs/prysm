package rpc

import (
	"context"
	"strings"
	"testing"

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
	if err != nil {
		t.Fatal(err)
	}
	ctxMD := map[string][]string{
		"authorization": {token},
	}
	ctx := context.Background()
	ctx = metadata.NewIncomingContext(ctx, ctxMD)
	_, err = interceptor(ctx, "xyz", unaryInfo, unaryHandler)
	if err != nil {
		t.Fatal(err)
	}
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
	if err != nil {
		t.Fatal(err)
	}
	ctxMD := map[string][]string{
		"authorization": {token},
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
