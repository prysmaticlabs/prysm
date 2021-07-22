package rpc

import (
	pb "github.com/prysmaticlabs/prysm/proto/prysm/v2"
)

var _ pb.AuthServer = (*Server)(nil)
