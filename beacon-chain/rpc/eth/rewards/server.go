package rewards

import (
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/blockfetcher"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/statefetcher"
)

type Server struct {
	BlockFetcher blockfetcher.Fetcher
	StateFetcher statefetcher.Fetcher
}
