package endtoend

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/multiformats/go-multiaddr"
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
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pborman/uuid"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	ev "github.com/prysmaticlabs/prysm/endtoend/evaluators"
	"github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"google.golang.org/grpc"
)

type end2EndConfig struct {
	minimalConfig  bool
	tmpPath        string
	epochsToRun    uint64
	numValidators  uint64
	numBeaconNodes uint64
	contractAddr   common.Address
	evaluators     []ev.Evaluator
}

type beaconNodeInfo struct {
	processID   int
	datadir     string
	rpcPort     uint64
	monitorPort uint64
	grpcPort    uint64
	multiAddr   string
}

type validatorClientInfo struct {
	processID   int
	monitorPort uint64
}

func runEndToEndTest(t *testing.T, config *end2EndConfig) {
	tmpPath := path.Join("/tmp/e2e/", uuid.NewRandom().String()[:18])
	if err := os.MkdirAll(tmpPath, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	config.tmpPath = tmpPath
	fmt.Printf("Test Path: %s\n", tmpPath)

	contractAddr, keystorePath, eth1PID := startEth1(t, tmpPath)
	config.contractAddr = contractAddr
	beaconNodes := startBeaconNodes(t, config)
	valClients := initializeValidators(t, config, keystorePath, beaconNodes)
	processIDs := []int{eth1PID}
	for _, vv := range valClients {
		processIDs = append(processIDs, vv.processID)
	}
	for _, bb := range beaconNodes {
		processIDs = append(processIDs, bb.processID)
	}
	defer logOutput(t, tmpPath)
	defer killProcesses(t, processIDs)

	beaconLogFile, err := os.Open(path.Join(tmpPath, "beacon-0.log"))
	if err != nil {
		t.Fatal(err)
	}
	if err := waitForTextInFile(beaconLogFile, "Chain started within the last epoch"); err != nil {
		t.Fatal(err)
	}

	if config.numBeaconNodes > 1 {
		t.Run("peers_connect", func(t *testing.T) {
			for _, bNode := range beaconNodes {
				if err := peersConnect(bNode.monitorPort, config.numBeaconNodes-1); err != nil {
					t.Fatalf("failed to connect to peers: %v", err)
				}
			}
		})
	}

	conn, err := grpc.Dial("127.0.0.1:4000", grpc.WithInsecure())
	if err != nil {
		t.Fatalf("fail to dial: %v", err)
	}
	beaconClient := eth.NewBeaconChainClient(conn)

	currentEpoch := uint64(0)
	for currentEpoch < config.epochsToRun {
		if currentEpoch > 0 {
			newEpochText := fmt.Sprintf("\"Starting next epoch\" epoch=%d", currentEpoch)
			if err := waitForTextInFile(beaconLogFile, newEpochText); err != nil {
				t.Fatal(err)
			}
		}

		for _, evaluator := range config.evaluators {
			// Only run if the policy says so.
			if !evaluator.Policy(currentEpoch) {
				continue
			}
			t.Run(fmt.Sprintf(evaluator.Name, currentEpoch), func(t *testing.T) {
				if err := evaluator.Evaluation(beaconClient); err != nil {
					t.Fatal(err)
				}
			})
		}
		currentEpoch++
	}

	if currentEpoch < config.epochsToRun {
		t.Fatalf("test ended prematurely, only reached epoch %d", currentEpoch)
	}
}

// startEth1 starts an eth1 local dev chain and deploys a deposit contract.
func startEth1(t *testing.T, tmpPath string) (common.Address, string, int) {
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
		"--dev.period=0",
	}
	cmd := exec.Command(binaryPath, args...)
	file, err := os.Create(path.Join(tmpPath, "eth1.log"))
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stdout = file
	cmd.Stderr = file
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start eth1 chain: %v", err)
	}

	if err = waitForTextInFile(file, "IPC endpoint opened"); err != nil {
		t.Fatal(err)
	}

	// Connect to the started geth dev chain.
	client, err := rpc.Dial(path.Join(tmpPath, "eth1data/geth.ipc"))
	if err != nil {
		t.Fatalf("failed to connect to ipc: %v", err)
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
	keystore, err := keystore.DecryptKey(jsonBytes, "" /*password*/)
	if err != nil {
		t.Fatal(err)
	}
	// Advancing the blocks eth1follow distance to prevent issues reading the chain.
	if err := mineBlocks(web3, keystore, params.BeaconConfig().Eth1FollowDistance); err != nil {
		t.Fatalf("unable to advance chain: %v", err)
	}

	txOpts, err := bind.NewTransactor(key, "" /*password*/)
	if err != nil {
		t.Fatal(err)
	}

	nonce, err := web3.PendingNonceAt(context.Background(), keystore.Address)
	if err != nil {
		t.Fatal(err)
	}
	txOpts.Nonce = big.NewInt(int64(nonce))
	contractAddr, tx, _, err := contracts.DeployDepositContract(txOpts, web3, txOpts.From)
	if err != nil {
		t.Fatalf("failed to deploy deposit contract: %v", err)
	}

	// Wait for contract to mine.
	for pending := true; pending; _, pending, err = web3.TransactionByHash(context.Background(), tx.Hash()) {
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	return contractAddr, keystorePath, cmd.Process.Pid
}

// startBeaconNodes starts the requested amount of beacon nodes, passing in the deposit contract given.
func startBeaconNodes(t *testing.T, config *end2EndConfig) []*beaconNodeInfo {
	numNodes := config.numBeaconNodes

	nodeInfo := []*beaconNodeInfo{}
	for i := uint64(0); i < numNodes; i++ {
		newNode := startNewBeaconNode(t, config, nodeInfo)
		nodeInfo = append(nodeInfo, newNode)
	}

	return nodeInfo
}

func startNewBeaconNode(t *testing.T, config *end2EndConfig, beaconNodes []*beaconNodeInfo) *beaconNodeInfo {
	tmpPath := config.tmpPath
	index := len(beaconNodes)
	binaryPath, found := bazel.FindBinary("beacon-chain", "beacon-chain")
	if !found {
		t.Log(binaryPath)
		t.Fatal("beacon chain binary not found")
	}
	file, err := os.Create(path.Join(tmpPath, fmt.Sprintf("beacon-%d.log", index)))
	if err != nil {
		t.Fatal(err)
	}

	args := []string{
		"--bootstrap-node=\"\"",
		"--no-discovery",
		"--http-web3provider=http://127.0.0.1:8545",
		"--web3provider=ws://127.0.0.1:8546",
		"--p2p-host-ip=127.0.0.1",
		fmt.Sprintf("--datadir=%s/eth2-beacon-node-%d", tmpPath, index),
		fmt.Sprintf("--deposit-contract=%s", config.contractAddr.Hex()),
		fmt.Sprintf("--rpc-port=%d", 4000+index),
		fmt.Sprintf("--p2p-udp-port=%d", 12000+index),
		fmt.Sprintf("--p2p-tcp-port=%d", 13000+index),
		fmt.Sprintf("--monitoring-port=%d", 8080+index),
		fmt.Sprintf("--grpc-gateway-port=%d", 3200+index),
	}

	if config.minimalConfig {
		args = append(args, "--minimal-config")
	}
	// After the first node is made, have all following nodes connect to all previously made nodes.
	if index >= 1 {
		for p := 0; p < index; p++ {
			args = append(args, fmt.Sprintf("--peer=%s", beaconNodes[p].multiAddr))
		}
	}

	t.Logf("Starting beacon chain with flags %s", strings.Join(args, " "))
	cmd := exec.Command(binaryPath, args...)
	cmd.Stderr = file
	cmd.Stdout = file
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start beacon node: %v", err)
	}

	if err = waitForTextInFile(file, "Connected to eth1 proof-of-work"); err != nil {
		t.Fatal(err)
	}

	response, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/p2p", 8080+index))
	if err != nil {
		t.Fatalf("failed to get p2p info: %v", err)
	}
	dataInBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	pageContent := string(dataInBytes)
	if err := response.Body.Close(); err != nil {
		t.Fatal(err)
	}
	addrPrefix := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/", 13000+index)
	startIdx := strings.Index(pageContent, addrPrefix)
	if startIdx == -1 {
		t.Fatalf("did not find peer text in %s", pageContent)
	}
	multiAddr, err := multiaddr.NewMultiaddr(pageContent[startIdx : startIdx+len(addrPrefix)+53])
	if err != nil {
		t.Fatal(err)
	}

	return &beaconNodeInfo{
		processID:   cmd.Process.Pid,
		datadir:     fmt.Sprintf("%s/eth2-beacon-node-%d", tmpPath, index),
		rpcPort:     (4000) + uint64(index),
		monitorPort: 8080 + uint64(index),
		grpcPort:    3200 + uint64(index),
		multiAddr:   multiAddr.String(),
	}
}

// initializeValidators sends the deposits to the eth1 chain and starts the validator clients.
func initializeValidators(
	t *testing.T,
	config *end2EndConfig,
	keystorePath string,
	beaconNodes []*beaconNodeInfo,
) []*validatorClientInfo {
	tmpPath := config.tmpPath
	contractAddress := config.contractAddr
	validatorNum := config.numValidators
	beaconNodeNum := config.numBeaconNodes
	binaryPath, found := bazel.FindBinary("validator", "validator")
	if !found {
		t.Fatal("validator binary not found")
	}

	if validatorNum%beaconNodeNum != 0 {
		t.Fatal("Validator count is not easily divisible by beacon node count.")
	}

	valClients := make([]*validatorClientInfo, beaconNodeNum)
	validatorsPerNode := validatorNum / beaconNodeNum
	for n := uint64(0); n < beaconNodeNum; n++ {
		file, err := os.Create(path.Join(tmpPath, fmt.Sprintf("vals-%d.log", n)))
		if err != nil {
			t.Fatal(err)
		}
		args := []string{
			fmt.Sprintf("--interop-num-validators=%d", validatorsPerNode),
			fmt.Sprintf("--interop-start-index=%d", validatorsPerNode*n),
			fmt.Sprintf("--monitoring-port=%d", 9080+n),
			fmt.Sprintf("--beacon-rpc-provider=localhost:%d", 4000+n),
		}
		cmd := exec.Command(binaryPath, args...)
		cmd.Stdout = file
		cmd.Stderr = file
		t.Logf("Starting validator client with flags %s", strings.Join(args, " "))
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		valClients[n] = &validatorClientInfo{
			processID:   cmd.Process.Pid,
			monitorPort: 9080 + n,
		}
	}

	for n := uint64(0); n < beaconNodeNum; n++ {
		file, err := os.Open(path.Join(tmpPath, fmt.Sprintf("vals-%d.log", n)))
		if err != nil {
			t.Fatal(err)
		}

		if err = waitForTextInFile(file, "Waiting for beacon chain start log"); err != nil {
			t.Fatal(err)
		}
	}

	client, err := rpc.Dial(path.Join(tmpPath, "eth1data/geth.ipc"))
	if err != nil {
		t.Fatal(err)
	}
	web3 := ethclient.NewClient(client)

	jsonBytes, err := ioutil.ReadFile(keystorePath)
	if err != nil {
		t.Fatal(err)
	}
	r := bytes.NewReader(jsonBytes)
	txOps, err := bind.NewTransactor(r, "" /*password*/)
	if err != nil {
		t.Fatal(err)
	}
	minDeposit := big.NewInt(int64(params.BeaconConfig().MaxEffectiveBalance))
	txOps.Value = minDeposit.Mul(minDeposit, big.NewInt(int64(params.BeaconConfig().GweiPerEth)))
	txOps.GasLimit = 4000000

	contract, err := contracts.NewDepositContract(contractAddress, web3)
	if err != nil {
		t.Fatal(err)
	}

	deposits, roots, _ := testutil.SetupInitialDeposits(t, validatorNum)
	var tx *types.Transaction
	for index, dd := range deposits {
		tx, err = contract.Deposit(txOps, dd.Data.PublicKey, dd.Data.WithdrawalCredentials, dd.Data.Signature, roots[index])
		if err != nil {
			t.Error("unable to send transaction to contract")
			continue
		}
	}

	// Wait for last tx to mine.
	for pending := true; pending; _, pending, err = web3.TransactionByHash(context.Background(), tx.Hash()) {
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	keystore, err := keystore.DecryptKey(jsonBytes, "" /*password*/)
	if err != nil {
		t.Fatal(err)
	}
	if err := mineBlocks(web3, keystore, 20); err != nil {
		t.Fatal(err)
	}

	return valClients
}

func peersConnect(port uint64, expectedPeers uint64) error {
	response, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/p2p", port))
	if err != nil {
		return err
	}
	dataInBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	pageContent := string(dataInBytes)
	if err := response.Body.Close(); err != nil {
		return err
	}
	fmt.Println(pageContent)
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

func mineBlocks(web3 *ethclient.Client, keystore *keystore.Key, blocksToMake uint64) error {
	nonce, err := web3.PendingNonceAt(context.Background(), keystore.Address)
	if err != nil {
		return err
	}
	chainID, err := web3.NetworkID(context.Background())
	if err != nil {
		return err
	}
	block, err := web3.BlockByNumber(context.Background(), nil)
	if err != nil {
		return err
	}
	finishBlock := block.NumberU64() + blocksToMake

	for block.NumberU64() <= finishBlock {
		spamTX := types.NewTransaction(nonce, keystore.Address, big.NewInt(0), 21000, big.NewInt(1e6), []byte{})
		signed, err := types.SignTx(spamTX, types.NewEIP155Signer(chainID), keystore.PrivateKey)
		if err != nil {
			return err
		}
		if err := web3.SendTransaction(context.Background(), signed); err != nil {
			return err
		}
		nonce++
		time.Sleep(250 * time.Microsecond)
		block, err = web3.BlockByNumber(context.Background(), nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func killProcesses(t *testing.T, pIDs []int) {
	for _, id := range pIDs {
		process, err := os.FindProcess(id)
		if err != nil {
			t.Fatalf("could not find process %d: %v", id, err)
		}
		if err := process.Kill(); err != nil {
			t.Fatal(err)
		}
	}
}

func logOutput(t *testing.T, tmpPath string) {
	if t.Failed() {
		t.Log("beacon-1.log")
		beacon1LogFile, err := os.Open(path.Join(tmpPath, "beacon-1.log"))
		if err != nil {
			t.Fatal(err)
		}
		scanner := bufio.NewScanner(beacon1LogFile)
		for scanner.Scan() {
			currentLine := scanner.Text()
			t.Log(currentLine)
		}
		t.Log("vals-1.log")
		vals1LogFile, err := os.Open(path.Join(tmpPath, "vals-1.log"))
		if err != nil {
			t.Fatal(err)
		}
		scanner = bufio.NewScanner(vals1LogFile)
		for scanner.Scan() {
			currentLine := scanner.Text()
			t.Log(currentLine)
		}
	}
}

func waitForTextInFile(file *os.File, text string) error {
	checks := 0
	maxChecks := int(4 * params.BeaconConfig().SecondsPerSlot * params.BeaconConfig().SlotsPerEpoch)
	// Put a limit on how many times we can check to prevent endless looping.
	for checks < maxChecks {
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
				return nil
			}
		}
		if err := scanner.Err(); err != nil {
			return err
		}
		checks++
	}
	return fmt.Errorf("could not find requested text %s in logs", text)
}
