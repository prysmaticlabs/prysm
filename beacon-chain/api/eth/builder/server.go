package builder

import (
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/api/lookup"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain"
)

type Server struct {
	FinalizationFetcher   blockchain.FinalizationFetcher
	OptimisticModeFetcher blockchain.OptimisticModeFetcher
	Stater                lookup.Stater
}
