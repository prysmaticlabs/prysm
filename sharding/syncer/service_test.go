package syncer

import "github.com/ethereum/go-ethereum/sharding"

var _ = sharding.Service(&Syncer{})
