package beaconv1

import ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1"

var _ ethpb.BeaconChainServer = (*Server)(nil)
