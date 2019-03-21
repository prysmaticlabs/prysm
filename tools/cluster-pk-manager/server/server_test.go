package main

import (
	pb "github.com/prysmaticlabs/prysm/proto/cluster"
)

var _ = pb.PrivateKeyServiceServer(&server{})
