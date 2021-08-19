package node

import (
	ethpbservice "github.com/prysmaticlabs/prysm/proto/eth/service"
)

var _ ethpbservice.BeaconNodeServer = (*Server)(nil)
