package syncer

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/mainchain"
	"github.com/ethereum/go-ethereum/sharding/p2p"
	"github.com/ethereum/go-ethereum/sharding/p2p/messages"
)

// RespondCollationBody is called by a node responding to another node's request
// for a collation body given a (shardID, chunkRoot, period, proposerAddress) tuple.
// The proposer will fetch the corresponding data from persistent storage (shardDB) by
// constructing a collation header from the input and calculating its hash.
func RespondCollationBody(req p2p.Message, signer mainchain.Signer, collationFetcher sharding.CollationFetcher) (*messages.CollationBodyResponse, error) {
	// Type assertion helps us catch incorrect data requests.
	msg, ok := req.Data.(messages.CollationBodyRequest)
	if !ok {
		return nil, fmt.Errorf("received incorrect data request type: %v", msg)
	}

	header := sharding.NewCollationHeader(msg.ShardID, msg.ChunkRoot, msg.Period, msg.Proposer, nil)
	sig, err := signer.Sign(header.Hash())
	if err != nil {
		return nil, fmt.Errorf("could not sign received header: %v", err)
	}

	// Adds the signature to the header before calculating the hash used for db lookups.
	header.AddSig(sig)

	// Fetch the collation by its header hash from the shardChainDB.
	headerHash := header.Hash()
	collation, err := collationFetcher.CollationByHeaderHash(&headerHash)
	if err != nil {
		return nil, fmt.Errorf("could not fetch collation: %v", err)
	}

	return &messages.CollationBodyResponse{HeaderHash: &headerHash, Body: collation.Body()}, nil
}

// RequestCollationBody fetches a collation header record submitted to the SMC for
// a shardID, period pair and constructs a p2p collationBodyRequest that will
// then be relayed to the appropriate proposer that submitted the collation header.
// In production, this will be done within a notary service.
func RequestCollationBody(fetcher mainchain.RecordFetcher, shardID *big.Int, period *big.Int) (*messages.CollationBodyRequest, error) {

	record, err := fetcher.CollationRecords(&bind.CallOpts{}, shardID, period)
	if err != nil {
		return nil, fmt.Errorf("could not fetch collation record from SMC: %v", err)
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

	return &messages.CollationBodyRequest{
		ChunkRoot: &chunkRoot,
		ShardID:   shardID,
		Period:    period,
		Proposer:  &record.Proposer,
	}, nil
}
