package rpc

import (
	"context"

	"github.com/dgrijalva/jwt-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// noAuthPaths keeps track of the paths which do not require
// authentication from our API.
var noAuthPaths = map[string]bool{
	"/proto/Login":  true,
	"/proto/Signup": true,
}

func (s *Server) ServerUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Skip authorize when the path doesn't require auth.
		if !noAuthPaths[info.FullMethod] {
			if err := s.authorize(ctx); err != nil {
				return nil, err
			}
		}

		h, err := handler(ctx, req)
		log.Debugf("Request - Method: %s, Error: %v\n", info.FullMethod, err)
		return h, err
	}
}

// Authorize verifies the token received is valid.
func (s *Server) authorize(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Errorf(codes.InvalidArgument, "retrieving metadata failed")
	}

	authHeader, ok := md["authorization"]
	if !ok {
		return status.Errorf(codes.Unauthenticated, "authorization token could not be found")
	}

	checkParsedKey := func(*jwt.Token) (interface{}, error) {
		return s.jwtKey, nil
	}
	token := authHeader[0]
	_, err := jwt.ParseWithClaims(token, &Claims{}, checkParsedKey)
	if err != nil {
		return err
	}
	return nil
}
