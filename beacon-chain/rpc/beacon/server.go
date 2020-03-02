package beacon

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
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// Server defines a server implementation of the gRPC Beacon Chain service,
// providing RPC endpoints to access data relevant to the Ethereum 2.0 phase 0
// beacon chain.
type Server struct {
	BeaconDB             db.ReadOnlyDatabase
	Ctx                  context.Context
	ChainStartFetcher    powchain.ChainStartFetcher
	HeadFetcher          blockchain.HeadFetcher
	FinalizationFetcher  blockchain.FinalizationFetcher
	ParticipationFetcher blockchain.ParticipationFetcher
	DepositFetcher       depositcache.DepositFetcher
	BlockFetcher         powchain.POWBlockFetcher
	GenesisTimeFetcher   blockchain.TimeFetcher
	StateNotifier        statefeed.Notifier
	BlockNotifier        blockfeed.Notifier
	AttestationNotifier  operation.Notifier
	AttestationsPool     attestations.Pool
	SlashingsPool        *slashings.Pool
	CanonicalStateChan   chan *pbp2p.BeaconState
	ChainStartChan       chan time.Time
}
