package proposer

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/p2p"
)

// simulateNotaryRequests simulates incoming p2p messages that will come from
// notaries once the system is in production.
func simulateNotaryRequests(shardID *big.Int, shardp2p sharding.ShardP2P) {
	// listens to headers submitted to the SMC and submits a request
	// for their body from the corresponding proposer.
	headerHash := "0x010201020"
	msg := &p2p.Message{
		Data: headerHash,
	}
	feed, err := shardp2p.Feed(sharding.CollationBodyRequest{})
	if err != nil {
		log.Error(fmt.Sprintf("Could not initialize p2p feed: %v", err))
		return
	}
	feed.Send(msg)
}
