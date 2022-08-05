package rpc

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// JWTInterceptor is a gRPC unary interceptor to authorize incoming requests.
func (s *Server) JWTInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if err := s.authorize(ctx); err != nil {
			return nil, err
		}
		h, err := handler(ctx, req)
		log.WithError(err).WithFields(logrus.Fields{
			"FullMethod": info.FullMethod,
			"Server":     info.Server,
		}).Debug("Request handled")
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
	return s.jwtSecret, nil
}
