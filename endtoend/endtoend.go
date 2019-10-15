package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os/exec"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
)

func main() {
	StartEth1()
}

// StartEth1 starts an eth1 local dev chain and deploys a deposit contract.
func StartEth1() {
	if err := exec.Command("rm", "-rf", "/tmp/eth1data").Run(); err != nil {
		panic(err)
	}

	args := []string{
		"--datadir=/tmp/eth1data",
		"--dev.period=4",
		"--unlock=0",
		"--password=password.txt",
		"--dev",
	}
	if err := exec.Command("geth", args...).Start(); err != nil {
		panic(err)
	}

	log.Println("started")
	time.Sleep(12 * time.Second)

	client, err := rpc.Dial("/tmp/eth1data/geth.ipc")
	if err != nil {
		panic(err)
	}
	web3 := ethclient.NewClient(client)
	log.Println("connected")

	fileName, err := exec.Command("ls", "/tmp/eth1data/keystore").Output()
	if err != nil {
		log.Fatal(err)
	}
	keystorePath := fmt.Sprintf("/tmp/eth1data/keystore/%s", strings.TrimSpace(string(fileName)))
	ks := keystore.NewKeyStore("/tmp/eth1data", keystore.StandardScryptN, keystore.StandardScryptP)
	jsonBytes, err := ioutil.ReadFile(keystorePath)
	if err != nil {
		log.Fatal(err)
	}

	password := ""
	account, err := ks.Import(jsonBytes, password, password)
	if err != nil {
		log.Fatal(err)
	}
	if err := ks.Unlock(account, ""); err != nil {
		panic(err)
	}
	fmt.Println(account.Address.Hex())

	nonce, err := web3.NonceAt(context.Background(), account.Address, nil)
	if err != nil {
		panic(err)
	}
	contractTx := types.NewContractCreation(
		nonce,
		big.NewInt(0),
		4000000,
		big.NewInt(10*1e9),
		common.Hex2Bytes(contracts.DepositContractBin[2:]),
	)

	signedTx, err := ks.SignTx(account, contractTx, big.NewInt(1337))
	if err != nil {
		panic(err)
	}

	if err := web3.SendTransaction(context.Background(), signedTx); err != nil {
		panic(err)
	}

	// Wait for contract to mine
	for pending := true; pending; _, pending, err = web3.TransactionByHash(context.Background(), signedTx.Hash()) {
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(4 * time.Second)
	}
	log.Print("Deposit contract deployed")

	contractReceipt, err := web3.TransactionReceipt(context.Background(), signedTx.Hash())
	if err != nil {
		panic(err)
	}
	log.Println(contractReceipt.ContractAddress.Hex())

	args = []string{
		"run",
		"//contracts/deposit-contract/sendDepositTx",
		"--",
		"--random-key",
		fmt.Sprintf("--keystoreUTCPath=%s", keystorePath),
		"--ipcPath=/tmp/eth1data/geth.ipc",
		fmt.Sprintf("--depositContract=%s", contractReceipt.ContractAddress.Hex()),
		"--depositAmount=3200000",
	}
	if err := exec.Command("bazel", args...).Start(); err != nil {
		panic(err)
	}
}
