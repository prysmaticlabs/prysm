// Package beaconv1 defines a gRPC beacon service implementation,
// following the official API standards https://ethereum.github.io/eth2.0-APIs/#/.
// This package includes the beacon and config endpoints.
package beaconv1

import (
	"context"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed/operation"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/voluntaryexits"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/statefetcher"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// Server defines a server implementation of the gRPC Beacon Chain service,
// providing RPC endpoints to access data relevant to the Ethereum 2.0 phase 0
// beacon chain.
type Server struct {
	BeaconDB            db.ReadOnlyDatabase
	Ctx                 context.Context
	ChainStartFetcher   powchain.ChainStartFetcher
	ChainInfoFetcher    blockchain.ChainInfoFetcher
	DepositFetcher      depositcache.DepositFetcher
	BlockFetcher        powchain.POWBlockFetcher
	GenesisTimeFetcher  blockchain.TimeFetcher
	BlockReceiver       blockchain.BlockReceiver
	StateNotifier       statefeed.Notifier
	BlockNotifier       blockfeed.Notifier
	AttestationNotifier operation.Notifier
	Broadcaster         p2p.Broadcaster
	AttestationsPool    attestations.Pool
	SlashingsPool       slashings.PoolManager
	VoluntaryExitsPool  voluntaryexits.PoolManager
	CanonicalStateChan  chan *pbp2p.BeaconState
	ChainStartChan      chan time.Time
	StateGenService     stategen.StateManager
	SyncChecker         sync.Checker
	StateFetcher        statefetcher.StateFetcher
}
