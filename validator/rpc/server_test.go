package rpc

import (
	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
)

var _ = pb.AuthServer(&Server{})
