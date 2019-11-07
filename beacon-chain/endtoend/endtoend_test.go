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
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pborman/uuid"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	"github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	prysmKeyStore "github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var log = logrus.WithField("prefix", "e2e")

type end2EndConfig struct {
	tmpPath        string
	minimalConfig  bool
	epochsToRun    uint64
	numValidators  uint64
	numBeaconNodes uint64
	evaluators     []evaluator
}

type beaconNodeInfo struct {
	processID   int
	datadir     string
	rpcPort     uint64
	monitorPort uint64
	grpcPort    uint64
	multiAddr   string
}

func TestEndToEnd_DemoConfig(t *testing.T) {
	tmpPath := path.Join("/tmp/e2e/", uuid.NewUUID().String()[:18])
	os.MkdirAll(tmpPath, os.ModePerm)
	fmt.Printf("Test Path: %s\n", tmpPath)

	demoConfig := &end2EndConfig{
		tmpPath:        tmpPath,
		minimalConfig:  false,
		epochsToRun:    8,
		numBeaconNodes: 2,
		numValidators:  8,
		evaluators: []evaluator{
			evaluator{
				name:       "activate_validators",
				policy:     onChainStart,
				evaluation: validatorsActivate,
			},
			evaluator{
				name:       "finalize_checkpoint",
				policy:     afterNEpochs(4),
				evaluation: finalizationOccurs,
			},
			// Evaluator{
			//	Name:       "validators_participate",
			// 	Policy:     AfterNEpochs(4),
			// 	Evaluation: ValidatorsParticipating,
			// },
		},
	}
	runEndToEndTest(t, demoConfig)
}

// func TestEndToEnd_MinimalConfig(t *testing.T) {
// 	tmpPath := path.Join("/tmp/e2e/", uuid.NewUUID().String()[:18])
// 	os.MkdirAll(tmpPath, os.ModePerm)
// 	fmt.Printf("Path for this test is %s\n", tmpPath)

// 	demoConfig := &End2EndConfig{
// 		NumBeaconNodes: 4,
// 		NumValidators:  64,
// 		MinimalConfig:  true,
// 		TmpPath:        tmpPath,
// 		Evaluators: []Evaluator{
// 			Evaluator{
// 				Name:       "activate_validators",
// 				Policy:     OnChainStart,
// 				Evaluation: ValidatorsActivate,
// 			},
// 			Evaluator{
// 				Name:       "finalize_checkpoint",
// 				Policy:     AfterNEpochs(4),
// 				Evaluation: FinalizationOccurs,
// 			},
// 			// Evaluator{
// 			//	Name:       "validators_participate",
// 			// 	Policy:     AfterNEpochs(4),
// 			// 	Evaluation: ValidatorsParticipating,
// 			// },
// 		},
// 	}
// 	RunEndToEndTest(t, demoConfig)
// }

func runEndToEndTest(t *testing.T, config *end2EndConfig) {
	tmpPath := config.tmpPath
	params.UseDemoBeaconConfig()
	contractAddr, keystorePath := StartEth1(t, tmpPath)
	beaconNodes := startBeaconNodes(t, config, contractAddr)
	InitializeValidators(t, tmpPath, contractAddr, keystorePath, beaconNodes, config.numValidators)

	beaconLogFile, err := os.Open(path.Join(tmpPath, "beacon-0.log"))
	if err != nil {
		t.Fatal(err)
	}
	if err := WaitForTextInFile(beaconLogFile, "Chain started within the last epoch"); err != nil {
		t.Fatal(err)
	}
	fmt.Println("Chain has started")

	t.Run("peers_connect", func(t *testing.T) {
		if err := PeersConnect(config.numBeaconNodes - 1); err != nil {
			t.Fatalf("failed to connect to peers: %v", err)
		}
	})

	conn, err := grpc.Dial("127.0.0.1:4000", grpc.WithInsecure())
	if err != nil {
		t.Fatalf("fail to dial: %v", err)
	}
	time.Sleep(2 * time.Second)
	beaconClient := eth.NewBeaconChainClient(conn)

	time.Sleep(time.Second * time.Duration(params.BeaconConfig().SecondsPerSlot))
	currentEpoch := uint64(0)
	// Run the evaluators for any in chainstart to execute.
	runEvaluators(t, beaconClient, config.evaluators)

	scanner := bufio.NewScanner(beaconLogFile)
	for scanner.Scan() && currentEpoch < config.epochsToRun {
		currentLine := scanner.Text()
		if strings.Contains(currentLine, "Finished applying state transition") {
			time.Sleep(time.Second * time.Duration(params.BeaconConfig().SecondsPerSlot))
			continue
		} else if !strings.Contains(currentLine, "Starting next epoch") {
			time.Sleep(time.Microsecond * 600)
			continue
		}

		// Only run evaluators when a new epoch is started.
		newEpochIndex := strings.Index(currentLine, "epoch=") + 6
		newEpoch, err := strconv.Atoi(currentLine[newEpochIndex : newEpochIndex+1])
		if err != nil {
			t.Fatalf("failed to convert logs to int: %v", err)
		}
		currentEpoch = uint64(newEpoch)
		fmt.Println("")
		fmt.Printf("Current Epoch: %d\n", currentEpoch)

		runEvaluators(t, beaconClient, config.evaluators)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}

	if currentEpoch < config.epochsToRun {
		t.Fatalf("test ended prematurely, only reached epoch %d", currentEpoch)
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
		"--dev.period=4",
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
	file, err := os.Create(path.Join(tmpPath, "eth1.log"))
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stdout = file
	cmd.Stderr = file
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	if err = WaitForTextInFile(file, "IPC endpoint opened"); err != nil {
		t.Fatal(err)
	}

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

	fmt.Printf("Contract deployed at %s\n", contractAddr.Hex())
	return contractAddr, keystorePath
}

// startBeaconNodes starts the requested amount of beacon nodes, passing in the deposit contract given.
func startBeaconNodes(t *testing.T, config *end2EndConfig, contractAddress common.Address) []*beaconNodeInfo {
	tmpPath := config.tmpPath
	numNodes := config.numBeaconNodes
	binaryPath, found := bazel.FindBinary("beacon-chain", "beacon-chain")
	if !found {
		t.Log(binaryPath)
		t.Fatal("beacon chain binary not found")
	}

	nodeInfo := make([]*beaconNodeInfo, numNodes)
	for i := uint64(0); i < numNodes; i++ {
		file, err := os.Create(path.Join(tmpPath, fmt.Sprintf("beacon-%d.log", i)))
		if err != nil {
			t.Fatal(err)
		}

		args := []string{
			"--no-discovery",
			"--http-web3provider=http://127.0.0.1:8545",
			"--web3provider=ws://127.0.0.1:8546",
			"--p2p-host-ip=127.0.0.1",
			fmt.Sprintf("--datadir=%s/eth2-beacon-node-%d", tmpPath, i),
			fmt.Sprintf("--deposit-contract=%s", contractAddress.Hex()),
			fmt.Sprintf("--rpc-port=%d", 4000+i),
			fmt.Sprintf("--p2p-tcp-port=%d", 13000+i),
			fmt.Sprintf("--monitoring-port=%d", 8080+i),
			fmt.Sprintf("--grpc-gateway-port=%d", 3200+i),
		}

		if config.minimalConfig {
			args = append(args, "--minimal-config")
		}
		// After the first node is made, have all following nodes connect to all previously made nodes.
		if i >= 1 {
			for p := uint64(0); p < i; p++ {
				args = append(args, fmt.Sprintf("--peer=%s", nodeInfo[p].multiAddr))
			}
		}

		cmd := exec.Command(binaryPath, args...)
		cmd.Stderr = file
		cmd.Stdout = file
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}

		if err = WaitForTextInFile(file, "Connected to eth1 proof-of-work"); err != nil {
			t.Fatal(err)
		}

		response, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/p2p", 8080+i))
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
		multiAddr := pageContent[startIdx : startIdx+85]

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
		}
	}

	for n := uint64(0); n < beaconNodeNum; n++ {
		file, err := os.Create(path.Join(tmpPath, fmt.Sprintf("vals%d.log", n)))
		if err != nil {
			t.Fatal(err)
		}
		args := []string{
			"--password=e2etest",
			fmt.Sprintf("--keystore-path=%s/valkeys%d/", tmpPath, n),
			fmt.Sprintf("--monitoring-port=%d", 9080+n),
			fmt.Sprintf("--beacon-rpc-provider=localhost:%d", 4000+n),
		}
		cmd := exec.Command(binaryPath, args...)
		cmd.Stdout = file
		cmd.Stderr = file
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}

		if err = WaitForTextInFile(file, "Waiting for beacon chain start log"); err != nil {
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

	var tx *types.Transaction
	for _, validatorKey := range validatorKeys {
		data, err := prysmKeyStore.DepositInput(validatorKey, validatorKey, 3200000000)
		if err != nil {
			log.Errorf("Could not generate deposit input data: %v", err)
			continue
		}

		tx, err = depositContract.Deposit(txOps, data.PublicKey, data.WithdrawalCredentials, data.Signature)
		if err != nil {
			log.Error("unable to send transaction to contract")
			continue
		}
	}

	// Wait for last tx to mine.
	for pending := true; pending; _, pending, err = web3.TransactionByHash(context.Background(), tx.Hash()) {
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(4 * time.Second)
	}

	// Sleep 5 ETH blocks.
	log.Printf("%d deposits mined", len(validatorKeys))
	time.Sleep(5 * 4 * time.Second)
}

func PeersConnect(expectedPeers uint64) error {
	response, err := http.Get("http://127.0.0.1:8080/p2p")
	if err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	dataInBytes, err := ioutil.ReadAll(response.Body)
	pageContent := string(dataInBytes)
	if err := response.Body.Close(); err != nil {
		return err
	}
	startIdx := strings.Index(pageContent, "peers") - 2
	peerCount, err := strconv.Atoi(pageContent[startIdx : startIdx+1])
	if err != nil {
		return err
	}
	if expectedPeers != uint64(peerCount) {
		return fmt.Errorf("unexpected amount of peers, expected %d, received %d", expectedPeers, peerCount)
	}
	return nil
}

func WaitForTextInFile(file *os.File, text string) error {
	found := false
	for !found {
		// Pass some time to not spam file checks.
		time.Sleep(2 * time.Second)
		// Rewind the file pointer to the start of the file so we can read it again.
		_, err := file.Seek(0, io.SeekStart)
		if err != nil {
			return fmt.Errorf("could not rewind file to start: %v", err)
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			currentLine := scanner.Text()
			if strings.Contains(currentLine, text) {
				found = true
				break
			}
		}
		if err := scanner.Err(); err != nil {
			return err
		}
	}
	return nil
}
