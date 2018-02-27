package sharding

import (
	"context"
	"fmt"
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding/contracts"
)

type collatorClient interface {
	Account() (*accounts.Account, error)
	ChainReader() ethereum.ChainReader
	VMCCaller() *contracts.VMCCaller
}

// processPendingTransactions fetches the pending tx's from the txpool
// in the geth node. Sorts by descending order of gasprice, remove txs
// that ask for too much gas adds a the tx into the collation and
// throws on failure
func processPendingTransactions(c collatorClient) error {
	txpool := NewTxPool(config TxPoolConfig, chainconfig *params.ChainConfig, chain blockChain)
	pendingTxs, err := txpool.Pending()
	if err != nil {
		return fmt.Errorf("Could not fetch pending tx's: %v", err)
	}
	for k, v := range pendingTxs {
		...
	}
}
