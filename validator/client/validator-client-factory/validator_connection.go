package validator_client_factory

import (
	"time"

	"google.golang.org/grpc"
)

type ValidatorConnection struct {
	GrpcClientConn *grpc.ClientConn
	BeaconApiConn  struct {
		Url     string
		Timeout time.Duration
	}
}
