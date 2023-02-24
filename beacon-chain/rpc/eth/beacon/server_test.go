package beacon

import ethpbservice "github.com/prysmaticlabs/prysm/v3/proto/eth/service"

var _ ethpbservice.BeaconChainServer = (*Server)(nil)
