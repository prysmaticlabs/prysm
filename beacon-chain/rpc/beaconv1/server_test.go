package beaconv1

import ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"

var _ ethpb.BeaconChainServer = (*Server)(nil)
