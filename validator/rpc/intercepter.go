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
		"/ethereum.eth.v1alpha1.Auth/Signup":                      true,
		"/ethereum.eth.v1alpha1.Auth/Login":                       true,
		"/ethereum.eth.v1alpha1.Auth/Logout":                      true,
		"/ethereum.eth.v1alpha1.Auth/HasUsedWeb":                  true,
		"/ethereum.eth.v1alpha1.Wallet/HasWallet":                 true,
		"/ethereum.eth.v1alpha1.Beacon/GetBeaconStatus":           true,
		"/ethereum.eth.v1alpha1.Beacon/GetValidatorParticipation": true,
		"/ethereum.eth.v1alpha1.Beacon/GetValidatorPerformance":   true,
		"/ethereum.eth.v1alpha1.Beacon/GetValidatorBalances":      true,
		"/ethereum.eth.v1alpha1.Beacon/GetValidators":             true,
		"/ethereum.eth.v1alpha1.Beacon/GetValidatorQueue":         true,
		"/ethereum.eth.v1alpha1.Beacon/GetPeers":                  true,
		"/ethereum.eth.v1alpha1.Beacon/StreamValidatorLogs":       true,
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
