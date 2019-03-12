package e2etestutil

import (
	"bytes"
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
)

var depositsForChainStart = big.NewInt(8)
var minDepositAmount = big.NewInt(1e9)
var maxDepositAmount = big.NewInt(32e9)
var chainStartDelay = big.NewInt(10)

func (g *GoEthereumInstance) DeployDepositContract() common.Address {
	client, err := g.node.Attach()
	if err != nil {
		g.t.Fatal(err)
	}
	eclient := ethclient.NewClient(client)
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
		eclient,
		depositsForChainStart,
		minDepositAmount,
		maxDepositAmount,
		chainStartDelay,
		common.HexToAddress("0x0"), // drain address
	)
	if err != nil {
		g.t.Fatal(err)
	}

	g.t.Log("Waiting for contract to be mined in proof-of-work chain")
	for pending := true; pending; _, pending, err = eclient.TransactionByHash(context.Background(), tx.Hash()) {
		if err != nil {
			g.t.Fatal(err)
		}
		time.Sleep(1 * time.Second)
	}

	g.depositContract = depContract
	g.DepositContractAddr = addr
	return addr
}

func (g *GoEthereumInstance) Deposit() {

}
