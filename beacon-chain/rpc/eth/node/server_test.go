package node

import (
	ethpbservice "github.com/prysmaticlabs/prysm/v3/proto/eth/service"
)

var _ ethpbservice.BeaconNodeServer = (*Server)(nil)
