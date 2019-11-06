package endtoend

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/pborman/uuid"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	"github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	prysmKeyStore "github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var log = logrus.WithField("prefix", "e2e")

type End2EndConfig struct {
	NumValidators  uint64
	NumBeaconNodes uint64
}

type beaconNodeInfo struct {
	processID   int
	datadir     string
	rpcPort     uint64
	monitorPort uint64
	grpcPort    uint64
	multiAddr   string
}

func TestEndToEnd(t *testing.T) {
	tmpPath := path.Join("/tmp/e2e/", uuid.NewUUID().String()[:18])
	os.MkdirAll(tmpPath, os.ModePerm)

	params.UseDemoBeaconConfig()
	contractAddr, keystorePath := StartEth1(t, tmpPath)
	beaconNodes := StartBeaconNodes(t, tmpPath, contractAddr, 1)
	InitializeValidators(t, tmpPath, contractAddr, keystorePath, beaconNodes, 8)

	time.Sleep(1 * time.Minute)

	conn, err := grpc.Dial("127.0.0.1:4000", grpc.WithInsecure())
	if err != nil {
		t.Fatalf("fail to dial: %v", err)
	}
	time.Sleep(1 * time.Second)
	beaconClient := eth.NewBeaconChainClient(conn)
	time.Sleep(1 * time.Second)

	for i := 0; i < 1; i++ {
		in := new(ptypes.Empty)
		chainHead, err := beaconClient.GetChainHead(context.Background(), in)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(chainHead.BlockSlot)

		// if AfterChainStart(chainHead) {
		// 	if err := ValidatorsActivate(beaconClient, 8); err != nil {
		// 		t.Fatal(err)
		// 	}
		// }

		// if AfterChainStart(chainHead) {
		// 	if err := ValidatorsActivate(beaconClient, 8); err != nil {
		// 		t.Fatal(err)
		// 	}
		// }

		time.Sleep(time.Second * 6)
	}
}

// StartEth1 starts an eth1 local dev chain and deploys a deposit contract.
func StartEth1(t *testing.T, tmpPath string) (common.Address, string) {
	binaryPath, found := bazel.FindBinary("cmd/geth", "geth")
	if !found {
		t.Fatal("go-ethereum binary not found")
	}

	args := []string{
		fmt.Sprintf("--datadir=%s", path.Join(tmpPath, "eth1data/")),
		"--rpc",
		"--rpcaddr=0.0.0.0",
		"--rpccorsdomain=\"*\"",
		"--rpcvhosts=\"*\"",
		"--ws",
		"--wsaddr=0.0.0.0",
		"--wsorigins=\"*\"",
		"--dev",
	}
	cmd := exec.Command(binaryPath, args...)
	file, err := os.Create(path.Join("/tmp", "eth1.log"))
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stdout = file
	cmd.Stderr = file
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(12 * time.Second)

	// Connect to the started geth dev chain.
	client, err := rpc.Dial(path.Join(tmpPath, "eth1data/geth.ipc"))
	if err != nil {
		t.Fatal(err)
	}
	web3 := ethclient.NewClient(client)

	// Access the dev account keystore to deploy the contract.
	fileName, err := exec.Command("ls", path.Join(tmpPath, "eth1data/keystore")).Output()
	if err != nil {
		t.Fatal(err)
	}
	keystorePath := path.Join(tmpPath, fmt.Sprintf("eth1data/keystore/%s", strings.TrimSpace(string(fileName))))
	jsonBytes, err := ioutil.ReadFile(keystorePath)
	if err != nil {
		t.Fatal(err)
	}
	key := bytes.NewReader(jsonBytes)

	txOpts, err := bind.NewTransactor(key, "")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Dev Account: %s\n", txOpts.From.Hex())
	minDeposit := big.NewInt(1e9)
	contractAddr, tx, _, err := contracts.DeployDepositContract(txOpts, web3, minDeposit, txOpts.From)
	if err != nil {
		t.Fatal(err)
	}

	// Wait for contract to mine.
	for pending := true; pending; _, pending, err = web3.TransactionByHash(context.Background(), tx.Hash()) {
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(4 * time.Second)
	}

	log.Printf("Contract deployed at %s\n", contractAddr.Hex())
	return contractAddr, keystorePath
}

// StartBeaconNodes starts the requested amount of beacon nodes, passing in the deposit contract given.
func StartBeaconNodes(t *testing.T, tmpPath string, contractAddress common.Address, numNodes uint64) []*beaconNodeInfo {
	binaryPath, found := bazel.FindBinary("beacon-chain", "beacon-chain")
	if !found {
		t.Log(binaryPath)
		t.Fatal("beacon chain binary not found")
	}

	nodeInfo := make([]*beaconNodeInfo, numNodes)
	for i := uint64(0); i < numNodes; i++ {
		args := []string{
			"--no-discovery",
			"--no-genesis-delay",
			"--http-web3provider=http://127.0.0.1:8545",
			"--web3provider=ws://127.0.0.1:8546",
			fmt.Sprintf("--datadir=%s/eth2-beacon-node-%d", tmpPath, i),
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

		cmd := exec.Command(binaryPath, args...)
		file, err := os.Create(path.Join(tmpPath, fmt.Sprintf("beacon-%d.log", i)))
		if err != nil {
			t.Fatal(err)
		}
		cmd.Stderr = file
		cmd.Stdout = file
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		time.Sleep(24 * time.Second)

		response, err := http.Get("http://127.0.0.1:8080/p2p")
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(2 * time.Second)

		// Get the response body as a string
		dataInBytes, err := ioutil.ReadAll(response.Body)
		pageContent := string(dataInBytes)
		if err := response.Body.Close(); err != nil {
			t.Fatal(err)
		}

		startIdx := strings.Index(pageContent, "self=") + 5
		multiAddr := pageContent[startIdx : startIdx+86]

		nodeInfo[i] = &beaconNodeInfo{
			processID:   cmd.Process.Pid,
			datadir:     fmt.Sprintf("%s/eth2-beacon-node-%d", tmpPath, i),
			rpcPort:     4000 + i,
			monitorPort: 8080 + i,
			grpcPort:    3200 + i,
			multiAddr:   multiAddr,
		}
	}

	return nodeInfo
}

// InitializeValidators sends the deposits to the eth1 chain and starts the validator clients.
func InitializeValidators(
	t *testing.T,
	tmpPath string,
	contractAddress common.Address,
	keystorePath string,
	beaconNodes []*beaconNodeInfo,
	validatorNum uint64,
) {
	binaryPath, found := bazel.FindBinary("validator", "validator")
	if !found {
		t.Fatal("validator binary not found")
	}

	beaconNodeNum := uint64(len(beaconNodes))
	if validatorNum%beaconNodeNum != 0 {
		t.Fatal("Validator count is not easily divisible by beacon node count.")
	}
	validatorsPerNode := validatorNum / beaconNodeNum
	for n := uint64(0); n < beaconNodeNum; n++ {
		for i := n * validatorsPerNode; i < (n+1)*validatorsPerNode; i++ {
			args := []string{
				"accounts",
				"create",
				"--password=e2etest",
				fmt.Sprintf("--keystore-path=%s/valkeys%d/", tmpPath, n),
			}
			if err := exec.Command(binaryPath, args...).Run(); err != nil {
				t.Fatal(err)
			}
			time.Sleep(4 * time.Second)
		}
	}
	log.Printf("%d accounts created\n", validatorNum)

	for n := uint64(0); n < beaconNodeNum; n++ {
		file, err := os.Create(path.Join(tmpPath, fmt.Sprintf("vals%d.log", n)))
		if err != nil {
			t.Fatal(err)
		}
		args := []string{
			"--password=e2etest",
			fmt.Sprintf("--keystore-path=%s/valkeys%d/", tmpPath, n),
			fmt.Sprintf("--monitoring-port=%d", 9080+n),
			fmt.Sprintf("--beacon-rpc-provider=localhost:%d", beaconNodes[n].rpcPort),
		}
		cmd := exec.Command(binaryPath, args...)
		cmd.Stdout = file
		cmd.Stderr = file
		time.Sleep(10 * time.Second)
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		log.Printf("%d Validators started for beacon node %d", validatorsPerNode, n)
	}

	client, err := rpc.Dial(path.Join(tmpPath, "eth1data/geth.ipc"))
	if err != nil {
		t.Fatal(err)
	}
	web3 := ethclient.NewClient(client)

	jsonBytes, err := ioutil.ReadFile(keystorePath)
	if err != nil {
		log.Fatal(err)
	}
	r := bytes.NewReader(jsonBytes)
	txOps, err := bind.NewTransactor(r, "")
	if err != nil {
		t.Fatal(err)
	}
	minDeposit := big.NewInt(3.2 * 1e9)
	txOps.Value = minDeposit.Mul(minDeposit, big.NewInt(1e9))
	txOps.GasLimit = 4000000

	depositContract, err := contracts.NewDepositContract(contractAddress, web3)
	if err != nil {
		log.Fatal(err)
	}

	validatorKeys := make(map[string]*prysmKeyStore.Key)
	for n := uint64(0); n < beaconNodeNum; n++ {
		prysmKeystorePath := path.Join(tmpPath, fmt.Sprintf("valkeys%d/", n))
		store := prysmKeyStore.NewKeystore(prysmKeystorePath)
		prefix := params.BeaconConfig().ValidatorPrivkeyFileName
		keysForNode, err := store.GetKeys(prysmKeystorePath, prefix, "e2etest")
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range keysForNode {
			validatorKeys[k] = v
		}
	}

	for _, validatorKey := range validatorKeys {
		data, err := prysmKeyStore.DepositInput(validatorKey, validatorKey, 3200000000)
		if err != nil {
			log.Errorf("Could not generate deposit input data: %v", err)
			continue
		}

		tx, err := depositContract.Deposit(txOps, data.PublicKey, data.WithdrawalCredentials, data.Signature)
		if err != nil {
			log.Error("unable to send transaction to contract")
			continue
		}

		// Wait for contract to mine.
		for pending := true; pending; _, pending, err = web3.TransactionByHash(context.Background(), tx.Hash()) {
			if err != nil {
				log.Fatal(err)
			}
			time.Sleep(4 * time.Second)
		}

		log.WithFields(logrus.Fields{
			"Transaction Hash": fmt.Sprintf("%#x", bytesutil.Trunc(tx.Hash().Bytes())),
			"Public Key":       fmt.Sprintf("%#x", bytesutil.Trunc(validatorKey.PublicKey.Marshal())),
		}).Info("Deposit mined")
	}
}

func printCMDLogs(stdout io.Reader, maxLines int) {
	// read command's stdout line by line
	in := bufio.NewScanner(stdout)

	// Sometimes get stuck so limiting at 25 lines
	lines := 0
	for in.Scan() && lines < maxLines {
		log.Printf(in.Text()) // write each line to your log, or anything you need
		lines++
	}
	if err := in.Err(); err != nil {
		log.Printf("error: %s", err)
	}
}
