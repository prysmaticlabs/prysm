package proposer

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/mainchain"
	"github.com/ethereum/go-ethereum/sharding/p2p"
)

func collationBodyResponse(req p2p.Message, signer mainchain.Signer, collationFetcher sharding.CollationFetcher) (*sharding.CollationBodyResponse, error) {
	// Type assertion helps us catch incorrect data requests.
	msg, ok := req.Data.(sharding.CollationBodyRequest)
	if !ok {
		return nil, fmt.Errorf("received incorrect data request type: %v", msg)
	}

	header := sharding.NewCollationHeader(msg.ShardID, msg.ChunkRoot, msg.Period, msg.Proposer, nil)
	sig, err := signer.Sign(header.Hash())
	if err != nil {
		return nil, fmt.Errorf("Could not sign received header: %v", err)
	}

	// Adds the signature to the header before calculating the hash used for db lookups.
	header.AddSig(sig)

	// Fetch the collation by its header hash from the shardChainDB.
	headerHash := header.Hash()
	collation, err := collationFetcher.CollationByHeaderHash(&headerHash)
	if err != nil {
		return nil, fmt.Errorf("could not fetch collation: %v", err)
	}

	return &sharding.CollationBodyResponse{HeaderHash: &headerHash, Body: collation.Body()}, nil
}

// simulateNotaryRequests simulates incoming p2p messages that will come from
// notaries once the system is in production.
func constructNotaryRequest(reader mainchain.Reader, caller mainchain.Caller, shardID *big.Int, periodLength int64) (*sharding.CollationBodyRequest, error) {

	blockNumber, err := reader.BlockByNumber(context.Background(), nil)
	if err != nil {
		return nil, fmt.Errorf("Could not fetch current block number: %v", err)
	}

	period := new(big.Int).Div(blockNumber.Number(), big.NewInt(periodLength))
	record, err := caller.SMCCaller().CollationRecords(&bind.CallOpts{}, shardID, period)
	if err != nil {
		return nil, fmt.Errorf("Could not fetch collation record from SMC: %v", err)
	}

	// Checks if we got an empty collation record. If the SMCCaller does not find
	// a collation header, it returns an array of [32]byte filled with 0's.
	// Better way to do this?
	sum := 0
	for _, val := range record.ChunkRoot {
		sum += int(val)
	}
	if sum == 0 {
		return nil, nil
	}

	// Converts from fixed size [32]byte to []byte slice.
	chunkRoot := common.BytesToHash(record.ChunkRoot[:])

	return &sharding.CollationBodyRequest{
		ChunkRoot: &chunkRoot,
		ShardID:   shardID,
		Period:    period,
		Proposer:  &record.Proposer,
	}, nil
}
