package p2p

import "github.com/ethereum/go-ethereum/sharding"

var _ = sharding.Service(&node{})
