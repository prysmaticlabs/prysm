package rpc

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/golang-jwt/jwt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// noAuthPaths keeps track of the paths which do not require
// authentication from our API.
var (
	noAuthPaths = map[string]bool{
		"/ethereum.validator.accounts.v2.Auth/Initialize":                  true,
		"/ethereum.validator.accounts.v2.Wallet/HasWallet":                 true,
		"/ethereum.validator.accounts.v2.Beacon/GetBeaconStatus":           true,
		"/ethereum.validator.accounts.v2.Beacon/GetValidatorParticipation": true,
		"/ethereum.validator.accounts.v2.Beacon/GetValidatorPerformance":   true,
		"/ethereum.validator.accounts.v2.Beacon/GetValidatorBalances":      true,
		"/ethereum.validator.accounts.v2.Beacon/GetValidators":             true,
		"/ethereum.validator.accounts.v2.Beacon/GetValidatorQueue":         true,
		"/ethereum.validator.accounts.v2.Beacon/GetPeers":                  true,
		"/ethereum.validator.accounts.v2.Beacon/StreamValidatorLogs":       true,
	}
	authLock sync.RWMutex
)

// JWTInterceptor is a gRPC unary interceptor to authorize incoming requests
// for methods that are NOT in the noAuthPaths configuration map.
func (s *Server) JWTInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Skip authorize when the path doesn't require auth.
		authLock.RLock()
		shouldAuthenticate := !noAuthPaths[info.FullMethod]
		authLock.RUnlock()
		if shouldAuthenticate {
			if err := s.authorize(ctx); err != nil {
				return nil, err
			}
		}

		h, err := handler(ctx, req)
		log.Debugf("Request - Method: %s, Error: %v\n", info.FullMethod, err)
		return h, err
	}
}

// Authorize the token received is valid.
func (s *Server) authorize(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Errorf(codes.InvalidArgument, "Retrieving metadata failed")
	}

	authHeader, ok := md["authorization"]
	if !ok {
		return status.Errorf(codes.Unauthenticated, "Authorization token could not be found")
	}
	if len(authHeader) < 1 || !strings.Contains(authHeader[0], "Bearer ") {
		return status.Error(codes.Unauthenticated, "Invalid auth header, needs Bearer {token}")
	}
	token := strings.Split(authHeader[0], "Bearer ")[1]
	_, err := jwt.Parse(token, s.validateJWT)
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "Could not parse JWT token: %v", err)
	}
	return nil
}

func (s *Server) validateJWT(token *jwt.Token) (interface{}, error) {
	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
		return nil, fmt.Errorf("unexpected JWT signing method: %v", token.Header["alg"])
	}
	return s.jwtKey, nil
}
