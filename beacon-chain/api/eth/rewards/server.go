package rewards

import (
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/api/lookup"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain"
)

type Server struct {
	Blocker               lookup.Blocker
	OptimisticModeFetcher blockchain.OptimisticModeFetcher
	FinalizationFetcher   blockchain.FinalizationFetcher
	TimeFetcher           blockchain.TimeFetcher
	Stater                lookup.Stater
	HeadFetcher           blockchain.HeadFetcher
	BlockRewardFetcher    BlockRewardsFetcher
}
