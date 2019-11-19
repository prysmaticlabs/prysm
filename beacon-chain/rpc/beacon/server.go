package beacon

import (
	"context"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

// Server defines a server implementation of the gRPC Beacon Chain service,
// providing RPC endpoints to access data relevant to the Ethereum 2.0 phase 0
// beacon chain.
type Server struct {
	BeaconDB            db.Database
	Ctx                 context.Context
	ChainStartFetcher   powchain.ChainStartFetcher
	HeadFetcher         blockchain.HeadFetcher
	FinalizationFetcher blockchain.FinalizationFetcher
	StateFeedListener   blockchain.ChainFeeds
	Pool                operations.Pool
	IncomingAttestation chan *ethpb.Attestation
	CanonicalStateChan  chan *pbp2p.BeaconState
	ChainStartChan      chan time.Time
}
