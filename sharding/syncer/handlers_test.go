package syncer

import (
	"bytes"
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/contracts"
	"github.com/ethereum/go-ethereum/sharding/p2p"
	"github.com/ethereum/go-ethereum/sharding/params"
	shardparams "github.com/ethereum/go-ethereum/sharding/params"
	"github.com/ethereum/go-ethereum/sharding/proposer"

	pb "github.com/ethereum/go-ethereum/sharding/p2p/proto"
)

var (
	key, _            = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	addr              = crypto.PubkeyToAddress(key.PublicKey)
	accountBalance, _ = new(big.Int).SetString("1001000000000000000000", 10)
)

// Mock client for testing proposer.
type mockNode struct {
	smc         *contracts.SMC
	t           *testing.T
	depositFlag bool
	Backend     *backends.SimulatedBackend
	BlockNumber int64
}

type faultySMCCaller struct{}

func (f *faultySMCCaller) CollationRecords(opts *bind.CallOpts, arg0 *big.Int, arg1 *big.Int) (struct {
	ChunkRoot [32]byte
	Proposer  common.Address
	IsElected bool
	Signature []byte
}, error) {
	res := new(struct {
		ChunkRoot [32]byte
		Proposer  common.Address
		IsElected bool
		Signature []byte
	})
	return *res, errors.New("error fetching collation record")
}

func (m *mockNode) CreateTXOpts(value *big.Int) (*bind.TransactOpts, error) {
	txOpts := transactOpts()
	txOpts.Value = value
	return txOpts, nil
}

func (m *mockNode) SMCTransactor() *contracts.SMCTransactor {
	return &m.smc.SMCTransactor
}

func (m *mockNode) SMCCaller() *contracts.SMCCaller {
	return &m.smc.SMCCaller
}

func (m *mockNode) GetShardCount() (int64, error) {
	shardCount, err := m.SMCCaller().ShardCount(&bind.CallOpts{})
	if err != nil {
		return 0, err
	}
	return shardCount.Int64(), nil
}

func (m *mockNode) Account() *accounts.Account {
	return &accounts.Account{Address: addr}
}

func (m *mockNode) WaitForTransaction(ctx context.Context, hash common.Hash, durationInSeconds time.Duration) error {
	m.CommitWithBlock()
	m.FastForward(1)
	return nil
}

func (m *mockNode) TransactionReceipt(hash common.Hash) (*types.Receipt, error) {
	return m.Backend.TransactionReceipt(context.Background(), hash)
}

func (m *mockNode) DepositFlag() bool {
	return m.depositFlag
}

func (m *mockNode) FastForward(p int) {
	for i := 0; i < p*int(shardparams.DefaultConfig.PeriodLength); i++ {
		m.CommitWithBlock()
	}
}

func (m *mockNode) CommitWithBlock() {
	m.Backend.Commit()
	m.BlockNumber = m.BlockNumber + 1
}

func (m *mockNode) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	return types.NewBlockWithHeader(&types.Header{Number: big.NewInt(m.BlockNumber)}), nil
}

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

func transactOpts() *bind.TransactOpts {
	return bind.NewKeyedTransactor(key)
}

func setup(t *testing.T) (*backends.SimulatedBackend, *contracts.SMC) {
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance}})
	_, _, smc, err := contracts.DeploySMC(transactOpts(), backend)
	if err != nil {
		t.Fatalf("Failed to deploy SMC contract: %v", err)
	}
	backend.Commit()
	return backend, smc
}

func TestCollationBodyResponse(t *testing.T) {

	proposerAddress := common.BytesToAddress([]byte{})
	chunkRoot := common.BytesToHash([]byte{})

	goodReq := pb.CollationBodyRequest{
		ChunkRoot:       chunkRoot.Bytes(),
		ShardId:         1,
		Period:          1,
		ProposerAddress: proposerAddress.Bytes(),
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

	if _, err := RespondCollationBody(badMsg, signer, fetcher); err == nil {
		t.Errorf("Incorrect request should throw error. Expecting pb.CollationBodyRequest{}, received: %v", incorrectReq)
	}

	if _, err := RespondCollationBody(goodMsg, faultySigner, fetcher); err == nil {
		t.Error("Faulty signer should cause function to throw error. no error thrown.")
	}

	if _, err := RespondCollationBody(goodMsg, signer, faultyFetcher); err == nil {
		t.Error("Faulty collatiom fetcher should cause function to throw error. no error thrown.")
	}

	shardID := new(big.Int).SetUint64(goodReq.ShardId)
	chunkRoot = common.BytesToHash(goodReq.ChunkRoot)
	period := new(big.Int).SetUint64(goodReq.Period)
	proposer := common.BytesToAddress(goodReq.ProposerAddress)

	header := sharding.NewCollationHeader(
		shardID,
		&chunkRoot,
		period,
		&proposer,
		[]byte{})
	body := []byte{}

	response, err := RespondCollationBody(goodMsg, signer, fetcher)
	if err != nil {
		t.Fatalf("Could not construct collation body response: %v", err)
	}

	if common.BytesToHash(response.HeaderHash).Hex() != header.Hash().Hex() {
		t.Errorf("Incorrect header hash received. want: %v, received: %v", header.Hash().Hex(), common.BytesToHash(response.HeaderHash).Hex())
	}

	if !bytes.Equal(response.Body, body) {
		t.Errorf("Incorrect collation body received. want: %v, received: %v", response.Body, body)
	}
}

func TestConstructNotaryRequest(t *testing.T) {

	backend, smc := setup(t)
	node := &mockNode{smc: smc, t: t, Backend: backend}

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
	if err := proposer.AddHeader(node, node, collation); err != nil {
		t.Fatalf("Failed to add header to SMC: %v", err)
	}

	backend.Commit()

	if _, err := RequestCollationBody(&faultySMCCaller{}, shardID, period); err == nil {
		t.Errorf("Expected error from RequestCollationBody when using faulty SMCCaller, got nil")
	}

	request, err := RequestCollationBody(node.SMCCaller(), shardID, period)
	if err != nil {
		t.Fatalf("Could not construct request: %v", err)
	}

	// fetching an inexistent shardID, period pair from the SMC will return a nil request.
	nilRequest, err := RequestCollationBody(node.SMCCaller(), big.NewInt(20), big.NewInt(20))
	if err != nil {
		t.Fatalf("Could not construct request: %v", err)
	}

	if nilRequest != nil {
		t.Errorf("constructNotaryRequest should return nil for an inexistent collation header. got: %v", err)
	}

	if common.BytesToHash(request.ChunkRoot).Hex() != chunkRoot.Hex() {
		t.Errorf("Chunk root from notary request incorrect. want: %v, got: %v", chunkRoot.Hex(), common.BytesToHash(request.ChunkRoot).Hex())
	}

	if common.BytesToHash(request.ProposerAddress).Hex() != proposerAddress.Hex() {
		t.Errorf("Proposer address from notary request incorrect. want: %v, got: %v", proposerAddress.Hex(), common.BytesToHash(request.ProposerAddress).Hex())
	}

	if shardID.Uint64() != request.ShardId {
		t.Errorf("ShardID from notary request incorrect. want: %d, got: %d", shardID.Uint64(), request.ShardId)
	}

	if request.Period != period.Uint64() {
		t.Errorf("Proposer address from notary request incorrect. want: %d, got: %d", period.Uint64(), request.Period)
	}
}
