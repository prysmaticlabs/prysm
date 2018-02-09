package sharding

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/sharding/contracts"

	"github.com/ethereum/go-ethereum/core/types"
)

type FakeCollatorClient struct {
	accountAccount *accounts.Account
	accountError   error
	chainReader    FakeChainReader
	contractCaller FakeContractCaller
}

func (c FakeCollatorClient) Account() (*accounts.Account, error) {
	return c.accountAccount, c.accountError
}

func (c FakeCollatorClient) ChainReader() ethereum.ChainReader {
	return c.chainReader
}

func (c FakeCollatorClient) VMCCaller() *contracts.VMCCaller {
	VMCCaller, err := contracts.NewVMCCaller(common.HexToAddress("0x0"), c.contractCaller)
	if err != nil {
		panic(err)
	}
	return VMCCaller
}

type FakeChainReader struct {
	subscribeNewHeadSubscription ethereum.Subscription
	subscribeNewHeadError        error
}

func (r FakeChainReader) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	return r.subscribeNewHeadSubscription, r.subscribeNewHeadError
}
func (r FakeChainReader) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return nil, nil
}
func (r FakeChainReader) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	return nil, nil
}
func (r FakeChainReader) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return nil, nil
}
func (r FakeChainReader) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	return nil, nil
}
func (r FakeChainReader) TransactionCount(ctx context.Context, blockHash common.Hash) (uint, error) {
	return 0, nil
}
func (r FakeChainReader) TransactionInBlock(ctx context.Context, blockHash common.Hash, index uint) (*types.Transaction, error) {
	return nil, nil
}

type FakeContractCaller struct {
	codeAtBytes       []byte
	codeAtError       error
	callContractBytes []byte
	callContractError error
}

func (c FakeContractCaller) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	return c.codeAtBytes, c.codeAtError
}

func (c FakeContractCaller) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	return c.callContractBytes, c.callContractError
}

func TestCheckShardsForProposal(t *testing.T) {
	tests := []struct {
		Name           string
		Head           *types.Header
		ExpectedPeriod *big.Int
		ExpectedError  string
		CollatorClient FakeCollatorClient
	}{
		{
			Name:          "collatorClient.Account should return an error",
			ExpectedError: "no account",
			CollatorClient: FakeCollatorClient{
				accountError: errors.New("no account"),
			},
		},
		{
			Name:          "VMCCaller.GetEligibleProposer should return an error",
			ExpectedError: "there is no cake",
			CollatorClient: FakeCollatorClient{
				accountAccount: &accounts.Account{},
				contractCaller: FakeContractCaller{
					callContractError: errors.New("there is no cake"),
				},
			},
			Head: &types.Header{Number: big.NewInt(100)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			if err := checkShardsForProposal(tt.CollatorClient, tt.Head); !strings.Contains(safeError(err), tt.ExpectedError) {
				t.Fatalf("Incorrect error! Wanted %v, got %v", tt.ExpectedError, err)
			}
		})
	}
}

func safeError(err error) string {
	if err != nil {
		return err.Error()
	}
	return "nil"
}
