package proposer

import (
	"bytes"
	"context"
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
type mockChainReader struct{}

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

func (m *mockChainReader) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	return types.NewBlock(&types.Header{
		Number: big.NewInt(params.DefaultConfig.PeriodLength),
	}, nil, nil, nil), nil
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
	mockReader := &mockChainReader{}

	// Fast forward to next period.
	for i := 0; i < int(params.DefaultConfig.PeriodLength); i++ {
		backend.Commit()
	}

	proposerAddress := common.BytesToAddress([]byte{})
	chunkRoot := common.BytesToHash([]byte{})
	header := sharding.NewCollationHeader(big.NewInt(0), &chunkRoot, big.NewInt(0), &proposerAddress, []byte{})
	collation := sharding.NewCollation(header, []byte{}, []*types.Transaction{})

	// Adds the header to the SMC.
	if err := addHeader(node, collation); err != nil {
		t.Fatalf("Failed to add header to SMC: %v", err)
	}

	backend.Commit()

	_, err := constructNotaryRequest(mockReader, node, big.NewInt(0), params.DefaultConfig.PeriodLength)
	if err != nil {
		t.Fatalf("Could not construct request: %v", err)
	}
}
