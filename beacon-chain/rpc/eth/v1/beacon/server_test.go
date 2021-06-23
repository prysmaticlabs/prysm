package beacon

import ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1"

var _ ethpb.BeaconChainServer = (*Server)(nil)
