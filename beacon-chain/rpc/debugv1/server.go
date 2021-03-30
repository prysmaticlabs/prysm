// Package beaconv1 defines a gRPC beacon service implementation,
// following the official API standards https://ethereum.github.io/eth2.0-APIs/#/.
// This package includes the beacon and config endpoints.
package debugv1

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/statefetcher"
)

// Server defines a server implementation of the gRPC Beacon Chain service,
// providing RPC endpoints to access data relevant to the Ethereum 2.0 phase 0
// beacon chain.
type Server struct {
	Ctx          context.Context
	BeaconDB     db.ReadOnlyDatabase
	StateFetcher statefetcher.Fetcher
}
