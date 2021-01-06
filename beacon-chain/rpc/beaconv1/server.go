// Package beaconv1 defines a gRPC beacon service implementation,
// following the official API standards https://ethereum.github.io/eth2.0-APIs/#/.
// This package includes the beacon and config endpoints.
package beaconv1

import (
	"context"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed/operation"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// Server defines a server implementation of the gRPC Beacon Chain service,
// providing RPC endpoints to access data relevant to the Ethereum 2.0 phase 0
// beacon chain.
// nolint: maligned
type Server struct {
	SlashingsPool       *slashings.Pool
	StateGen            *stategen.State
	ChainStartChan      chan time.Time
	CanonicalStateChan  chan *pbp2p.BeaconState
	BeaconDB            db.ReadOnlyDatabase
	BlockNotifier       blockfeed.Notifier
	AttestationsPool    attestations.Pool
	Broadcaster         p2p.Broadcaster
	AttestationNotifier operation.Notifier
	Ctx                 context.Context
	StateNotifier       statefeed.Notifier
	BlockReceiver       blockchain.BlockReceiver
	GenesisTimeFetcher  blockchain.TimeFetcher
	BlockFetcher        powchain.POWBlockFetcher
	DepositFetcher      depositcache.DepositFetcher
	ChainInfoFetcher    blockchain.ChainInfoFetcher
	ChainStartFetcher   powchain.ChainStartFetcher
	SyncChecker         sync.Checker
	ethpb.UnimplementedBeaconChainServer
}
