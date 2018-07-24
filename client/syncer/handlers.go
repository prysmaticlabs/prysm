package syncer

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/client/mainchain"
	"github.com/prysmaticlabs/prysm/client/types"
	pb "github.com/prysmaticlabs/prysm/proto/sharding/v1"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "syncer")

// RespondCollationBody is called by a node responding to another node's request
// for a collation body given a (shardID, chunkRoot, period, proposerAddress) tuple.
// The proposer will fetch the corresponding data from persistent storage (shardDB) by
// constructing a collation header from the input and calculating its hash.
func RespondCollationBody(req p2p.Message, collationFetcher types.CollationFetcher) (*pb.CollationBodyResponse, error) {
	// Type assertion helps us catch incorrect data requests.
	msg, ok := req.Data.(*pb.CollationBodyRequest)
	if !ok {
		log.Debugf("Request data type: %T", req.Data)
		return nil, fmt.Errorf("received incorrect data request type. Data: %+v", msg)
	}

	shardID := new(big.Int).SetUint64(msg.ShardId)
	chunkRoot := common.BytesToHash(msg.ChunkRoot)
	period := new(big.Int).SetUint64(msg.Period)
	proposer := common.BytesToAddress(msg.ProposerAddress)
	var sig [32]byte
	if len(msg.Signature) >= 32 {
		copy(sig[:], msg.Signature[0:32])
	}
	header := types.NewCollationHeader(shardID, &chunkRoot, period, &proposer, sig)

	// Fetch the collation by its header hash from the shardChainDB.
	headerHash := header.Hash()
	collation, err := collationFetcher.CollationByHeaderHash(&headerHash)
	if err != nil {
		return nil, fmt.Errorf("could not fetch collation: %v", err)
	}
	if collation == nil {
		return nil, nil
	}

	return &pb.CollationBodyResponse{HeaderHash: headerHash.Bytes(), Body: collation.Body()}, nil
}

// RequestCollationBody fetches a collation header record submitted to the SMC for
// a shardID, period pair and constructs a p2p collationBodyRequest that will
// then be relayed to the appropriate proposer that submitted the collation header.
// In production, this will be done within a notary service.
func RequestCollationBody(fetcher mainchain.RecordFetcher, shardID *big.Int, period *big.Int) (*pb.CollationBodyRequest, error) {

	record, err := fetcher.CollationRecords(&bind.CallOpts{}, shardID, period)
	if err != nil {
		return nil, fmt.Errorf("could not fetch collation record from SMC: %v", err)
	}

	sum := 0
	for _, val := range record.ChunkRoot {
		sum += int(val)
	}

	if sum == 0 {
		log.Debugf("No collation exists for shard %d and period %d", shardID, period)
		return nil, nil
	}

	// Converts from fixed size [32]byte to []byte slice.
	chunkRoot := common.BytesToHash(record.ChunkRoot[:])

	return &pb.CollationBodyRequest{
		ChunkRoot:       chunkRoot.Bytes(),
		ShardId:         shardID.Uint64(),
		Period:          period.Uint64(),
		ProposerAddress: record.Proposer.Bytes(),
	}, nil
}
