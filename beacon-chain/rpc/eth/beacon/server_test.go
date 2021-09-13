package beacon

import ethpbservice "github.com/prysmaticlabs/prysm/proto/eth/service"

var _ ethpbservice.BeaconChainServer = (*Server)(nil)
