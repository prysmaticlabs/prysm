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

type End2EndConfig struct {
	NumValidators  uint64
	NumBeaconNodes uint64
}

type beaconNodeInfo struct {
	processID   uint64
	datadir     string
	rpcPort     uint64
	monitorPort uint64
	grpcPort    uint64
	multiAddr   string
}

func main() {
	contractAddr, keystorePath := StartEth1()
	InitializeValidators(contractAddr, keystorePath)
}

// StartEth1 starts an eth1 local dev chain and deploys a deposit contract.
func StartEth1() (common.Address, string) {
	if err := exec.Command("rm", "-rf", "/tmp/eth1data").Run(); err != nil {
		panic(err)
	}
	args := []string{
		"--datadir=/tmp/eth1data",
		"--dev.period=4",
		"--unlock=0",
		"--password=password.txt",
		"--rpc",
		"--dev",
	}
	if err := exec.Command("geth", args...).Start(); err != nil {
		panic(err)
	}
	time.Sleep(12 * time.Second)

	// Connect to the started geth dev chain.
	client, err := rpc.Dial("/tmp/eth1data/geth.ipc")
	if err != nil {
		panic(err)
	}
	web3 := ethclient.NewClient(client)

	// Access the dev account keystore to deploy the contract.
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

	// Deploy the contract from the dev account.
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

	// Wait for contract to mine.
	for pending := true; pending; _, pending, err = web3.TransactionByHash(context.Background(), signedTx.Hash()) {
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(4 * time.Second)
	}

	// Retrieve the contract address from the TX receipt.
	contractReceipt, err := web3.TransactionReceipt(context.Background(), signedTx.Hash())
	if err != nil {
		panic(err)
	}
	log.Printf("Contract deployed at %s\n", contractReceipt.ContractAddress.Hex())

	return contractReceipt.ContractAddress, keystorePath
}

// StartBeaconNodes starts the requested amount of beacon nodes, passing in the deposit contract given.
func StartBeaconNodes(contractAddress common.Address, numNodes uint64) {
	nodeInfo := make([]*beaconNodeInfo, numNodes)
	for i := uint64(0); i < numNodes; i++ {
		args := []string{
			"run",
			"//beacon-chain",
			"--",
			// --no-custom-config using this?
			"--clear-db",
			"--http-web3provider=localhost:8545",
			"--web3provider=/tmp/eth1data/geth.ipc",
			fmt.Sprintf("--datadir=/tmp/eth2-beacon-node-%d", i),
			fmt.Sprintf("--deposit-contract=%s", contractAddress.Hex()),
			fmt.Sprintf("--rpc-port=%d", 4000+i),
			fmt.Sprintf("--monitoring-port=%d", 8080+i),
			fmt.Sprintf("--grpc-gateway-port=%d", 3200+i),
		}
		// After the first node is made, have all following nodes connect to all previously made nodes.
		if i >= 1 {
			for p := uint64(0); p < i-1; p++ {
				args = append(args, fmt.Sprintf("--peer=%s", nodeInfo[p].multiAddr))
			}
		}
		if err := exec.Command("bazel", args...).Start(); err != nil {
			panic(err)
		}

		nodeInfo[i] = &beaconNodeInfo{
			processID:   i,
			datadir:     fmt.Sprintf("/tmp/eth2-beacon-node-%d", i),
			rpcPort:     4000 + i,
			monitorPort: 8080 + i,
			grpcPort:    3200 + i,
			multiAddr:   "hahahaha",
		}
	}
}

// InitializeValidators sends the deposits to the eth1 chain and starts the validator clients.
func InitializeValidators(contractAddress common.Address, keystorePath string) {
	args := []string{
		"run",
		"//contracts/deposit-contract/sendDepositTx",
		"--",
		fmt.Sprintf("--prysm-keystore=%s", keystorePath),
		fmt.Sprintf("--keystoreUTCPath=%s", keystorePath),
		"--ipcPath=/tmp/eth1data/geth.ipc",
		fmt.Sprintf("--depositContract=%s", contractAddress.Hex()),
		"--depositAmount=3200000",
	}
	if err := exec.Command("bazel", args...).Start(); err != nil {
		panic(err)
	}
}
