package beacon

import (
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/blockchain"
	beacondb "github.com/prysmaticlabs/prysm/v5/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/rpc/lookup"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/sync"
)

type Server struct {
	SyncChecker           sync.Checker
	HeadFetcher           blockchain.HeadFetcher
	TimeFetcher           blockchain.TimeFetcher
	OptimisticModeFetcher blockchain.OptimisticModeFetcher
	CanonicalHistory      *stategen.CanonicalHistory
	BeaconDB              beacondb.ReadOnlyDatabase
	Stater                lookup.Stater
	ChainInfoFetcher      blockchain.ChainInfoFetcher
	FinalizationFetcher   blockchain.FinalizationFetcher
}
