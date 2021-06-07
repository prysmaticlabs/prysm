// Package debugv1 defines a gRPC beacon service implementation,
// following the official API standards https://ethereum.github.io/eth2.0-APIs/#/.
// This package includes the beacon and config endpoints.
package eventsv1

import (
	"context"

	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	opfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/operation"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
)

// Server defines a server implementation of the gRPC Beacon Chain service,
// providing RPC endpoints to access data relevant to the Ethereum 2.0 phase 0
// beacon chain.
type Server struct {
	Ctx               context.Context
	StateNotifier     statefeed.Notifier
	BlockNotifier     blockfeed.Notifier
	OperationNotifier opfeed.Notifier
}
