// Package internal provides a MockClient for testing.
package internal

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
)

// MockClient for testing proposer.
type MockClient struct {
	T           *testing.T
	Backend     *backends.SimulatedBackend
	BlockNumber int64
}

// TransactionReceipt returns the transaction receipt from the mock backend.
func (m *MockClient) TransactionReceipt(hash common.Hash) (*gethTypes.Receipt, error) {
	return m.Backend.TransactionReceipt(context.Background(), hash)
}

// CreateTXOpts returns transaction opts with the value assigned.
func (m *MockClient) CreateTXOpts(value *big.Int) (*bind.TransactOpts, error) {
	txOpts := TransactOpts()
	txOpts.Value = value
	return txOpts, nil
}

// TransactOpts Creates a new transaction options.
func TransactOpts() *bind.TransactOpts {
	return bind.NewKeyedTransactor(key)
}
