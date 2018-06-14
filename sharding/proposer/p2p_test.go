package proposer

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/p2p"
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
	return nil, nil
}

func (f *faultyCollationFetcher) CollationByHeaderHash(headerHash *common.Hash) (*sharding.Collation, error) {
	return nil, errors.New("could not fetch collation")
}

func TestCollationBodyResponse(t *testing.T) {
	incorrectRequest := faultyRequest{}
	signer := &mockSigner{}
	faultySigner := &faultySigner{}
	fetcher := &mockCollationFetcher{}
	faultyFetcher := &faultyCollationFetcher{}

	proposerAddress := common.BytesToAddress([]byte{})
	chunkRoot := common.BytesToHash([]byte{})

	badMsg := p2p.Message{
		Peer: p2p.Peer{},
		Data: incorrectRequest,
	}

	goodMsg := p2p.Message{
		Peer: p2p.Peer{},
		Data: &sharding.CollationBodyRequest{
			ChunkRoot: &chunkRoot,
			ShardID:   big.NewInt(1),
			Period:    big.NewInt(1),
			Proposer:  &proposerAddress,
		},
	}

	if _, err := collationBodyResponse(badMsg, signer, fetcher); err == nil {
		t.Errorf("Incorrect request should throw error. Expecting sharding.CollationBodyRequest{}, received: %v", incorrectRequest)
	}

	if _, err := collationBodyResponse(goodMsg, faultySigner, fetcher); err == nil {
		t.Error("Faulty signer should cause function to throw error. no error thrown.")
	}

	if _, err := collationBodyResponse(goodMsg, signer, faultyFetcher); err == nil {
		t.Error("Faulty collatiom fetcher should cause function to throw error. no error thrown.")
	}

}
