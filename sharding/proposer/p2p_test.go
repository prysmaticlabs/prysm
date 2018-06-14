package proposer

import (
	"bytes"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/p2p"
	"github.com/ethereum/go-ethereum/sharding/params"
)

type faultyRequest struct{}
type faultySigner struct{}
type faultyCollationFetcher struct{}

type mockSigner struct{}
type mockCollationFetcher struct{}

func (m *mockSigner) Sign(hash common.Hash) ([]byte, error) {
	return []byte{}, nil
}

func (f *faultySigner) Sign(hash common.Hash) ([]byte, error) {
	return []byte{}, errors.New("could not sign hash")
}

func (m *mockCollationFetcher) CollationByHeaderHash(headerHash *common.Hash) (*sharding.Collation, error) {
	shardID := big.NewInt(1)
	chunkRoot := common.BytesToHash([]byte{})
	period := big.NewInt(1)
	proposerAddress := common.BytesToAddress([]byte{})

	header := sharding.NewCollationHeader(shardID, &chunkRoot, period, &proposerAddress, []byte{})
	return sharding.NewCollation(header, []byte{}, []*types.Transaction{}), nil
}

func (f *faultyCollationFetcher) CollationByHeaderHash(headerHash *common.Hash) (*sharding.Collation, error) {
	return nil, errors.New("could not fetch collation")
}

func TestCollationBodyResponse(t *testing.T) {

	proposerAddress := common.BytesToAddress([]byte{})
	chunkRoot := common.BytesToHash([]byte{})

	goodReq := sharding.CollationBodyRequest{
		ChunkRoot: &chunkRoot,
		ShardID:   big.NewInt(1),
		Period:    big.NewInt(1),
		Proposer:  &proposerAddress,
	}
	incorrectReq := faultyRequest{}

	signer := &mockSigner{}
	faultySigner := &faultySigner{}
	fetcher := &mockCollationFetcher{}
	faultyFetcher := &faultyCollationFetcher{}

	badMsg := p2p.Message{
		Peer: p2p.Peer{},
		Data: incorrectReq,
	}

	goodMsg := p2p.Message{
		Peer: p2p.Peer{},
		Data: goodReq,
	}

	if _, err := collationBodyResponse(badMsg, signer, fetcher); err == nil {
		t.Errorf("Incorrect request should throw error. Expecting sharding.CollationBodyRequest{}, received: %v", incorrectReq)
	}

	if _, err := collationBodyResponse(goodMsg, faultySigner, fetcher); err == nil {
		t.Error("Faulty signer should cause function to throw error. no error thrown.")
	}

	if _, err := collationBodyResponse(goodMsg, signer, faultyFetcher); err == nil {
		t.Error("Faulty collatiom fetcher should cause function to throw error. no error thrown.")
	}

	header := sharding.NewCollationHeader(goodReq.ShardID, goodReq.ChunkRoot, goodReq.Period, goodReq.Proposer, []byte{})
	body := []byte{}

	response, err := collationBodyResponse(goodMsg, signer, fetcher)
	if err != nil {
		t.Fatalf("Could not construct collation body response: %v", err)
	}

	if response.HeaderHash.Hex() != header.Hash().Hex() {
		t.Errorf("Incorrect header hash received. want: %v, received: %v", header.Hash().Hex(), response.HeaderHash.Hex())
	}

	if !bytes.Equal(response.Body, body) {
		t.Errorf("Incorrect collation body received. want: %v, received: %v", response.Body, body)
	}
}

func TestConstructNotaryRequest(t *testing.T) {

	backend, smc := setup(t)
	node := &mockNode{smc: smc, t: t, backend: backend}

	// Fast forward to next period.
	for i := 0; i < int(params.DefaultConfig.PeriodLength); i++ {
		backend.Commit()
	}

	shardID := big.NewInt(0)
	period := big.NewInt(1)

	// We set the proposer address to the address used to setup the backend.
	proposerAddress := addr
	chunkRoot := common.BytesToHash([]byte("chunkroottest"))
	header := sharding.NewCollationHeader(shardID, &chunkRoot, period, &addr, []byte{})
	collation := sharding.NewCollation(header, []byte{}, []*types.Transaction{})

	// Adds the header to the SMC.
	if err := addHeader(node, collation); err != nil {
		t.Fatalf("Failed to add header to SMC: %v", err)
	}

	backend.Commit()

	// constructNotaryRequest reads from the deployed SMC and fetches a
	// collation record for the shardID, period pair. Then, it constructs a request
	// that will be broadcast over p2p and handled by a proposer that submitted the collation
	// header to the SMC in the first place.
	request, err := constructNotaryRequest(node, shardID, period)
	if err != nil {
		t.Fatalf("Could not construct request: %v", err)
	}

	// fetching an inexistent shardID, period pair from the SMC will return a nil request from a notary.
	nilRequest, err := constructNotaryRequest(node, big.NewInt(20), big.NewInt(20))
	if err != nil {
		t.Fatalf("Could not construct request: %v", err)
	}

	if nilRequest != nil {
		t.Errorf("constructNotaryRequest should return nil for an inexistent collation header. got: %v", err)
	}

	if request.ChunkRoot.Hex() != chunkRoot.Hex() {
		t.Errorf("Chunk root from notary request incorrect. want: %v, got: %v", chunkRoot.Hex(), request.ChunkRoot.Hex())
	}
	if request.Proposer.Hex() != proposerAddress.Hex() {
		t.Errorf("Proposer address from notary request incorrect. want: %v, got: %v", proposerAddress.Hex(), request.Proposer.Hex())
	}
	if request.ShardID.Cmp(shardID) != 0 {
		t.Errorf("ShardID from notary request incorrect. want: %s, got: %s", shardID, request.ShardID)
	}
	if request.Period.Cmp(period) != 0 {
		t.Errorf("Proposer address from notary request incorrect. want: %s, got: %s", period, request.Period)
	}
}
