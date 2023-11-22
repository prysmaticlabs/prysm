package lightclient

import (
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/api/lookup"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain"
)

type Server struct {
	Blocker     lookup.Blocker
	Stater      lookup.Stater
	HeadFetcher blockchain.HeadFetcher
}
