package proposer

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/mainchain"
	"github.com/ethereum/go-ethereum/sharding/p2p"
)

// simulateNotaryRequests simulates incoming p2p messages that will come from
// notaries once the system is in production.
func simulateNotaryRequests(client *mainchain.SMCClient, shardp2p *p2p.Server, shardID *big.Int) {
	for {
		blockNumber, err := client.ChainReader().BlockByNumber(context.Background(), nil)
		if err != nil {
			log.Error(fmt.Sprintf("Could not fetch current block number: %v", err))
			continue
		}
		period := new(big.Int).Div(blockNumber.Number(), big.NewInt(sharding.PeriodLength))
		record, err := client.SMCCaller().CollationRecords(&bind.CallOpts{}, shardID, period)
		if err != nil {
			log.Error(fmt.Sprintf("Could not fetch collation record from SMC: %v", err))
			continue
		}

		// Checks if we got an empty collation record. If the SMCCaller does not find
		// a collation header, it returns an array of [32]byte filled with 0's.
		sum := 0
		for _, val := range record.ChunkRoot {
			sum += int(val)
		}
		if sum == 0 {
			continue
		}

		// Converts from fixed size [32]byte to []byte slice.
		chunkRoot := common.BytesToHash(record.ChunkRoot[:])

		request := sharding.CollationBodyRequest{
			ChunkRoot: &chunkRoot,
			ShardID:   shardID,
			Period:    period,
			Proposer:  &record.Proposer,
		}
		msg := p2p.Message{
			Peer: nil,
			Data: request,
		}
		feed, err := shardp2p.Feed(sharding.CollationBodyRequest{})
		if err != nil {
			log.Error(fmt.Sprintf("Could not initialize p2p feed: %v", err))
			continue
		}
		feed.Send(msg)
		log.Info("Notary Simulator: sent request for collation body via a shardp2p feed")
		time.Sleep(5 * time.Second)
	}
}
