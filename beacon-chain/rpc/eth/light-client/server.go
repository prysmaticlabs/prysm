package lightclient

import (
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/lookup"
)

type Server struct {
	BeaconDB    db.ReadOnlyDatabase
	Stater      lookup.Stater
	HeadFetcher blockchain.HeadFetcher
}
