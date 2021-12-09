package components

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	cmdshared "github.com/prysmaticlabs/prysm/cmd"
	"github.com/prysmaticlabs/prysm/cmd/validator/flags"
	"github.com/prysmaticlabs/prysm/config/features"
	"github.com/prysmaticlabs/prysm/config/params"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/testing/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/testing/endtoend/params"
	e2etypes "github.com/prysmaticlabs/prysm/testing/endtoend/types"
	"github.com/prysmaticlabs/prysm/testing/util"
)

const depositGasLimit = 4000000

var _ e2etypes.ComponentRunner = (*ValidatorNode)(nil)
var _ e2etypes.ComponentRunner = (*ValidatorNodeSet)(nil)

// ValidatorNodeSet represents set of validator nodes.
type ValidatorNodeSet struct {
	e2etypes.ComponentRunner
	config  *e2etypes.E2EConfig
	started chan struct{}
}

// NewValidatorNodeSet creates and returns a set of validator nodes.
func NewValidatorNodeSet(config *e2etypes.E2EConfig) *ValidatorNodeSet {
	return &ValidatorNodeSet{
		config:  config,
		started: make(chan struct{}, 1),
	}
}

// Start starts the configured amount of validators, also sending and mining their deposits.
func (s *ValidatorNodeSet) Start(ctx context.Context) error {
	// Always using genesis count since using anything else would be difficult to test for.
	validatorNum := int(params.BeaconConfig().MinGenesisActiveValidatorCount)
	beaconNodeNum := e2e.TestParams.BeaconNodeCount
	if validatorNum%beaconNodeNum != 0 {
		return errors.New("validator count is not easily divisible by beacon node count")
	}
	validatorsPerNode := validatorNum / beaconNodeNum

	// Create validator nodes.
	nodes := make([]e2etypes.ComponentRunner, beaconNodeNum)
	for i := 0; i < beaconNodeNum; i++ {
		nodes[i] = NewValidatorNode(s.config, validatorsPerNode, i, validatorsPerNode*i)
	}

	// Wait for all nodes to finish their job (blocking).
	// Once nodes are ready passed in handler function will be called.
	return helpers.WaitOnNodes(ctx, nodes, func() {
		// All nodes stated, close channel, so that all services waiting on a set, can proceed.
		close(s.started)
	})
}

// Started checks whether validator node set is started and all nodes are ready to be queried.
func (s *ValidatorNodeSet) Started() <-chan struct{} {
	return s.started
}

// ValidatorNode represents a validator node.
type ValidatorNode struct {
	e2etypes.ComponentRunner
	config       *e2etypes.E2EConfig
	started      chan struct{}
	validatorNum int
	index        int
	offset       int
}

// NewValidatorNode creates and returns a validator node.
func NewValidatorNode(config *e2etypes.E2EConfig, validatorNum, index, offset int) *ValidatorNode {
	return &ValidatorNode{
		config:       config,
		validatorNum: validatorNum,
		index:        index,
		offset:       offset,
		started:      make(chan struct{}, 1),
	}
}

// Start starts a validator client.
func (v *ValidatorNode) Start(ctx context.Context) error {
	var pkg, target string
	if v.config.UsePrysmShValidator {
		pkg = ""
		target = "prysm_sh"
	} else {
		pkg = "cmd/validator"
		target = "validator"
	}
	binaryPath, found := bazel.FindBinary(pkg, target)
	if !found {
		return errors.New("validator binary not found")
	}

	config, validatorNum, index, offset := v.config, v.validatorNum, v.index, v.offset
	beaconRPCPort := e2e.TestParams.BeaconNodeRPCPort + index
	if beaconRPCPort >= e2e.TestParams.BeaconNodeRPCPort+e2e.TestParams.BeaconNodeCount {
		// Point any extra validator clients to a node we know is running.
		beaconRPCPort = e2e.TestParams.BeaconNodeRPCPort
	}

	file, err := helpers.DeleteAndCreateFile(e2e.TestParams.LogPath, fmt.Sprintf(e2e.ValidatorLogFileName, index))
	if err != nil {
		return err
	}
	gFile, err := helpers.GraffitiYamlFile(e2e.TestParams.TestPath)
	if err != nil {
		return err
	}
	args := []string{
		fmt.Sprintf("--%s=%s/eth2-val-%d", cmdshared.DataDirFlag.Name, e2e.TestParams.TestPath, index),
		fmt.Sprintf("--%s=%s", cmdshared.LogFileName.Name, file.Name()),
		fmt.Sprintf("--%s=%s", flags.GraffitiFileFlag.Name, gFile),
		fmt.Sprintf("--%s=%d", flags.InteropNumValidators.Name, validatorNum),
		fmt.Sprintf("--%s=%d", flags.InteropStartIndex.Name, offset),
		fmt.Sprintf("--%s=%d", flags.MonitoringPortFlag.Name, e2e.TestParams.ValidatorMetricsPort+index),
		fmt.Sprintf("--%s=%d", flags.GRPCGatewayPort.Name, e2e.TestParams.ValidatorGatewayPort+index),
		fmt.Sprintf("--%s=localhost:%d", flags.BeaconRPCProviderFlag.Name, beaconRPCPort),
		fmt.Sprintf("--%s=%s", flags.GrpcHeadersFlag.Name, "dummy=value,foo=bar"), // Sending random headers shouldn't break anything.
		fmt.Sprintf("--%s=%s", cmdshared.VerbosityFlag.Name, "debug"),
		"--" + cmdshared.ForceClearDB.Name,
		"--" + cmdshared.E2EConfigFlag.Name,
		"--" + cmdshared.AcceptTosFlag.Name,
	}
	// Only apply e2e flags to the current branch. New flags may not exist in previous release.
	if !v.config.UsePrysmShValidator {
		args = append(args, features.E2EValidatorFlags...)
	}
	args = append(args, config.ValidatorFlags...)

	if v.config.UsePrysmShValidator {
		args = append([]string{"validator"}, args...)
		log.Warning("Using latest release validator via prysm.sh")
	}

	cmd := exec.CommandContext(ctx, binaryPath, args...) // #nosec G204 -- Safe

	// Write stdout and stderr to log files.
	stdout, err := os.Create(path.Join(e2e.TestParams.LogPath, fmt.Sprintf("validator_%d_stdout.log", index)))
	if err != nil {
		return err
	}
	stderr, err := os.Create(path.Join(e2e.TestParams.LogPath, fmt.Sprintf("validator_%d_stderr.log", index)))
	if err != nil {
		return err
	}
	defer func() {
		if err := stdout.Close(); err != nil {
			log.WithError(err).Error("Failed to close stdout file")
		}
		if err := stderr.Close(); err != nil {
			log.WithError(err).Error("Failed to close stderr file")
		}
	}()
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	log.Infof("Starting validator client %d with flags: %s %s", index, binaryPath, strings.Join(args, " "))
	if err = cmd.Start(); err != nil {
		return err
	}

	// Mark node as ready.
	close(v.started)

	return cmd.Wait()
}

// Started checks whether validator node is started and ready to be queried.
func (v *ValidatorNode) Started() <-chan struct{} {
	return v.started
}

// SendAndMineDeposits sends the requested amount of deposits and mines the chain after to ensure the deposits are seen.
func SendAndMineDeposits(keystorePath string, validatorNum, offset int, partial bool) error {
	client, err := rpc.DialHTTP(fmt.Sprintf("http://127.0.0.1:%d", e2e.TestParams.Eth1RPCPort))
	if err != nil {
		return err
	}
	defer client.Close()
	web3 := ethclient.NewClient(client)

	keystoreBytes, err := ioutil.ReadFile(keystorePath) // #nosec G304
	if err != nil {
		return err
	}
	if err = sendDeposits(web3, keystoreBytes, validatorNum, offset, partial); err != nil {
		return err
	}
	mineKey, err := keystore.DecryptKey(keystoreBytes, "" /*password*/)
	if err != nil {
		return err
	}
	if err = mineBlocks(web3, mineKey, params.BeaconConfig().Eth1FollowDistance); err != nil {
		return fmt.Errorf("failed to mine blocks %w", err)
	}
	return nil
}

// sendDeposits uses the passed in web3 and keystore bytes to send the requested deposits.
func sendDeposits(web3 *ethclient.Client, keystoreBytes []byte, num, offset int, partial bool) error {
	txOps, err := bind.NewTransactorWithChainID(bytes.NewReader(keystoreBytes), "" /*password*/, big.NewInt(1337))
	if err != nil {
		return err
	}
	txOps.GasLimit = depositGasLimit
	txOps.Context = context.Background()
	nonce, err := web3.PendingNonceAt(context.Background(), txOps.From)
	if err != nil {
		return err
	}
	txOps.Nonce = big.NewInt(int64(nonce))

	contract, err := contracts.NewDepositContract(e2e.TestParams.ContractAddress, web3)
	if err != nil {
		return err
	}

	balances := make([]uint64, num+offset)
	for i := 0; i < len(balances); i++ {
		if i < len(balances)/2 && partial {
			balances[i] = params.BeaconConfig().MaxEffectiveBalance / 2
		} else {
			balances[i] = params.BeaconConfig().MaxEffectiveBalance
		}
	}
	deposits, trie, err := util.DepositsWithBalance(balances)
	if err != nil {
		return err
	}
	allDeposits := deposits
	allRoots := trie.Items()
	allBalances := balances
	if partial {
		deposits2, trie2, err := util.DepositsWithBalance(balances)
		if err != nil {
			return err
		}
		allDeposits = append(deposits, deposits2[:len(balances)/2]...)
		allRoots = append(trie.Items(), trie2.Items()[:len(balances)/2]...)
		allBalances = append(balances, balances[:len(balances)/2]...)
	}
	for index, dd := range allDeposits {
		if index < offset {
			continue
		}
		depositInGwei := big.NewInt(int64(allBalances[index]))
		txOps.Value = depositInGwei.Mul(depositInGwei, big.NewInt(int64(params.BeaconConfig().GweiPerEth)))
		_, err = contract.Deposit(txOps, dd.Data.PublicKey, dd.Data.WithdrawalCredentials, dd.Data.Signature, bytesutil.ToBytes32(allRoots[index]))
		if err != nil {
			return errors.Wrap(err, "unable to send transaction to contract")
		}
		txOps.Nonce = txOps.Nonce.Add(txOps.Nonce, big.NewInt(1))
	}
	return nil
}
