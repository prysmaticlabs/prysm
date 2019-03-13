package e2etestutil

import (
	"bytes"
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
)

var depositsForChainStart = big.NewInt(8)
var minDepositAmount = big.NewInt(1e9)  // gwei
var maxDepositAmount = big.NewInt(32e9) // gwei
var chainStartDelay = big.NewInt(10)    // seconds

func (g *GoEthereumInstance) DeployDepositContract() common.Address {
	keyjson, err := g.ks.Export(g.ks.Accounts()[0], "", "")
	if err != nil {
		g.t.Fatal(err)
	}
	txOpts, err := bind.NewTransactor(bytes.NewReader(keyjson), "")
	if err != nil {
		g.t.Fatal(err)
	}
	addr, tx, depContract, err := contracts.DeployDepositContract(
		txOpts,
		g.client,
		depositsForChainStart,
		minDepositAmount,
		maxDepositAmount,
		chainStartDelay,
		common.HexToAddress("0x0"), // drain address
	)
	if err != nil {
		g.t.Fatal(err)
	}

	g.waitForTransaction(tx)
	g.depositContract = depContract
	g.DepositContractAddr = addr
	return addr
}

func (g *GoEthereumInstance) SendValidatorDeposits(depositData [][]byte) {
	txOpts := g.txOpts()
	txOpts.Value = big.NewInt(0).Mul(maxDepositAmount, big.NewInt(1e9)) // wei

	if len(depositData) == 0 {
		g.t.Fatal("No deposit data provided to SendValidatorDeposits")
	}

	nonce, err := g.client.PendingNonceAt(context.Background(), txOpts.From)
	if err != nil {
		g.t.Fatal(err)
	}

	for _, depositDatum := range depositData {
		txOpts.Nonce = big.NewInt(int64(nonce))
		tx, err := g.depositContract.Deposit(txOpts, depositDatum)
		if err != nil {
			g.t.Logf("Failing data: %#x", depositDatum)
			g.t.Fatal(err)
		}
		g.t.Logf("Deposited %#x with nonce %d", tx.Hash(), txOpts.Nonce)

		nonce++
	}

	g.t.Logf("Sent %d deposits", len(depositData))

	g.flushTransactions()
}

// flushTransactions in the geth instance. For some reason, sending
// transactions to the go-ethereum instance leaves 2 transactions that haven't
// emit the log yet.
func (g *GoEthereumInstance) flushTransactions() {
	nonce, err := g.client.PendingNonceAt(context.Background(), g.ks.Accounts()[0].Address)
	if err != nil {
		g.t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		tx := types.NewTransaction(nonce, g.ks.Accounts()[0].Address, nil, 21000, nil, nil)
		tx, err = g.ks.SignTxWithPassphrase(g.ks.Accounts()[0], "", tx, big.NewInt(1337))
		if err != nil {
			g.t.Fatal(err)
		}
		if err := g.client.SendTransaction(context.Background(), tx); err != nil {
			g.t.Fatal(err)
		}
		nonce++
	}
}

func (g *GoEthereumInstance) txOpts() *bind.TransactOpts {
	keyjson, err := g.ks.Export(g.ks.Accounts()[0], "", "")
	if err != nil {
		g.t.Fatal(err)
	}
	txOpts, err := bind.NewTransactor(bytes.NewReader(keyjson), "")
	if err != nil {
		g.t.Fatal(err)
	}
	return txOpts
}

func (g *GoEthereumInstance) waitForTransaction(tx *types.Transaction) {
	// Send another transaction and wait for it to no longer be pending
	g.t.Log("Waiting for transaction to be mined in proof-of-work chain")
	var err error
	for pending := true; pending; _, pending, err = g.client.TransactionByHash(context.Background(), tx.Hash()) {
		if err != nil {
			g.t.Fatal(err)
		}
		time.Sleep(1 * time.Second)
	}
}
