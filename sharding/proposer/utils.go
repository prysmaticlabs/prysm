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

		// Converts from fixed size [32]byte to []byte slice.
		chunkRoot := common.BytesToHash(record.ChunkRoot[:])
		header := sharding.NewCollationHeader(shardID, &chunkRoot, period, &record.Proposer, []byte{})
		msg := &p2p.Message{
			Data: header.Hash(),
		}
		feed, err := shardp2p.Feed(sharding.CollationBodyRequest{})
		if err != nil {
			log.Error(fmt.Sprintf("Could not initialize p2p feed: %v", err))
			continue
		}
		feed.Send(msg)
		time.Sleep(time.Second)
	}
}
