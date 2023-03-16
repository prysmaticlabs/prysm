package rewards

import (
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/blockfetcher"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stategen"
)

type Server struct {
	BlockFetcher          blockfetcher.Fetcher
	OptimisticModeFetcher blockchain.OptimisticModeFetcher
	FinalizationFetcher   blockchain.FinalizationFetcher
	ReplayerBuilder       stategen.ReplayerBuilder
}
