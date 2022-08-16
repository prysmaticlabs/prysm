package components

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	cmdshared "github.com/prysmaticlabs/prysm/v3/cmd"
	"github.com/prysmaticlabs/prysm/v3/cmd/validator/flags"
	"github.com/prysmaticlabs/prysm/v3/config/features"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	validator_service_config "github.com/prysmaticlabs/prysm/v3/config/validator/service"
	contracts "github.com/prysmaticlabs/prysm/v3/contracts/deposit"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v3/io/file"
	"github.com/prysmaticlabs/prysm/v3/runtime/interop"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/components/eth1"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/v3/testing/endtoend/params"
	e2etypes "github.com/prysmaticlabs/prysm/v3/testing/endtoend/types"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
)

const depositGasLimit = 4000000
const DefaultFeeRecipientAddress = "0x099FB65722e7b2455043bfebF6177f1D2E9738d9"

var _ e2etypes.ComponentRunner = (*ValidatorNode)(nil)
var _ e2etypes.ComponentRunner = (*ValidatorNodeSet)(nil)
var _ e2etypes.MultipleComponentRunners = (*ValidatorNodeSet)(nil)

// ValidatorNodeSet represents set of validator nodes.
type ValidatorNodeSet struct {
	e2etypes.ComponentRunner
	config  *e2etypes.E2EConfig
	started chan struct{}
	nodes   []e2etypes.ComponentRunner
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
	prysmBeaconNodeNum := e2e.TestParams.BeaconNodeCount
	beaconNodeNum := prysmBeaconNodeNum + e2e.TestParams.LighthouseBeaconNodeCount
	if validatorNum%beaconNodeNum != 0 {
		return errors.New("validator count is not easily divisible by beacon node count")
	}
	validatorsPerNode := validatorNum / beaconNodeNum
	// Create validator nodes.
	nodes := make([]e2etypes.ComponentRunner, prysmBeaconNodeNum)
	for i := 0; i < prysmBeaconNodeNum; i++ {
		nodes[i] = NewValidatorNode(s.config, validatorsPerNode, i, validatorsPerNode*i)
	}
	s.nodes = nodes

	// Wait for all nodes to finish their job (blocking).
	// Once nodes are ready passed in handler function will be called.
	return helpers.WaitOnNodes(ctx, nodes, func() {
		// All nodes started, close channel, so that all services waiting on a set, can proceed.
		close(s.started)
	})
}

// Started checks whether validator node set is started and all nodes are ready to be queried.
func (s *ValidatorNodeSet) Started() <-chan struct{} {
	return s.started
}

// Pause pauses the component and its underlying process.
func (s *ValidatorNodeSet) Pause() error {
	for _, n := range s.nodes {
		if err := n.Pause(); err != nil {
			return err
		}
	}
	return nil
}

// Resume resumes the component and its underlying process.
func (s *ValidatorNodeSet) Resume() error {
	for _, n := range s.nodes {
		if err := n.Resume(); err != nil {
			return err
		}
	}
	return nil
}

// Stop stops the component and its underlying process.
func (s *ValidatorNodeSet) Stop() error {
	for _, n := range s.nodes {
		if err := n.Stop(); err != nil {
			return err
		}
	}
	return nil
}

// PauseAtIndex pauses the component and its underlying process at the desired index.
func (s *ValidatorNodeSet) PauseAtIndex(i int) error {
	if i >= len(s.nodes) {
		return errors.Errorf("provided index exceeds slice size: %d >= %d", i, len(s.nodes))
	}
	return s.nodes[i].Pause()
}

// ResumeAtIndex resumes the component and its underlying process at the desired index.
func (s *ValidatorNodeSet) ResumeAtIndex(i int) error {
	if i >= len(s.nodes) {
		return errors.Errorf("provided index exceeds slice size: %d >= %d", i, len(s.nodes))
	}
	return s.nodes[i].Resume()
}

// StopAtIndex stops the component and its underlying process at the desired index.
func (s *ValidatorNodeSet) StopAtIndex(i int) error {
	if i >= len(s.nodes) {
		return errors.Errorf("provided index exceeds slice size: %d >= %d", i, len(s.nodes))
	}
	return s.nodes[i].Stop()
}

// ComponentAtIndex returns the component at the provided index.
func (s *ValidatorNodeSet) ComponentAtIndex(i int) (e2etypes.ComponentRunner, error) {
	if i >= len(s.nodes) {
		return nil, errors.Errorf("provided index exceeds slice size: %d >= %d", i, len(s.nodes))
	}
	return s.nodes[i], nil
}

// ValidatorNode represents a validator node.
type ValidatorNode struct {
	e2etypes.ComponentRunner
	config       *e2etypes.E2EConfig
	started      chan struct{}
	validatorNum int
	index        int
	offset       int
	cmd          *exec.Cmd
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
	validatorHexPubKeys := make([]string, 0)
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
	beaconRPCPort := e2e.TestParams.Ports.PrysmBeaconNodeRPCPort + index
	if beaconRPCPort >= e2e.TestParams.Ports.PrysmBeaconNodeRPCPort+e2e.TestParams.BeaconNodeCount {
		// Point any extra validator clients to a node we know is running.
		beaconRPCPort = e2e.TestParams.Ports.PrysmBeaconNodeRPCPort
	}

	file, err := helpers.DeleteAndCreateFile(e2e.TestParams.LogPath, fmt.Sprintf(e2e.ValidatorLogFileName, index))
	if err != nil {
		return err
	}
	gFile, err := helpers.GraffitiYamlFile(e2e.TestParams.TestPath)
	if err != nil {
		return err
	}

	_, pubs, err := interop.DeterministicallyGenerateKeys(uint64(offset), uint64(validatorNum))
	if err != nil {
		return err
	}
	for _, pub := range pubs {
		validatorHexPubKeys = append(validatorHexPubKeys, hexutil.Encode(pub.Marshal()))
	}
	proposerSettingsPathPath, err := createProposerSettingsPath(validatorHexPubKeys, index)
	if err != nil {
		return err
	}
	args := []string{
		fmt.Sprintf("--%s=%s/eth2-val-%d", cmdshared.DataDirFlag.Name, e2e.TestParams.TestPath, index),
		fmt.Sprintf("--%s=%s", cmdshared.LogFileName.Name, file.Name()),
		fmt.Sprintf("--%s=%s", flags.GraffitiFileFlag.Name, gFile),
		fmt.Sprintf("--%s=%d", flags.MonitoringPortFlag.Name, e2e.TestParams.Ports.ValidatorMetricsPort+index),
		fmt.Sprintf("--%s=%d", flags.GRPCGatewayPort.Name, e2e.TestParams.Ports.ValidatorGatewayPort+index),
		fmt.Sprintf("--%s=localhost:%d", flags.BeaconRPCProviderFlag.Name, beaconRPCPort),
		fmt.Sprintf("--%s=%s", flags.GrpcHeadersFlag.Name, "dummy=value,foo=bar"), // Sending random headers shouldn't break anything.
		fmt.Sprintf("--%s=%s", cmdshared.VerbosityFlag.Name, "debug"),
		fmt.Sprintf("--%s=%s", flags.ProposerSettingsFlag.Name, proposerSettingsPathPath),
		"--" + cmdshared.ForceClearDB.Name,
		"--" + cmdshared.E2EConfigFlag.Name,
		"--" + cmdshared.AcceptTosFlag.Name,
	}
	// Only apply e2e flags to the current branch. New flags may not exist in previous release.
	if !v.config.UsePrysmShValidator {
		args = append(args, features.E2EValidatorFlags...)
	}
	if v.config.UseWeb3RemoteSigner {
		args = append(args, fmt.Sprintf("--%s=http://localhost:%d", flags.Web3SignerURLFlag.Name, Web3RemoteSignerPort))
		// Write the pubkeys as comma separated hex strings with 0x prefix.
		// See: https://docs.teku.consensys.net/en/latest/HowTo/External-Signer/Use-External-Signer/
		args = append(args, fmt.Sprintf("--%s=%s", flags.Web3SignerPublicValidatorKeysFlag.Name, strings.Join(validatorHexPubKeys, ",")))
	} else {
		// When not using remote key signer, use interop keys.
		args = append(args,
			fmt.Sprintf("--%s=%d", flags.InteropNumValidators.Name, validatorNum),
			fmt.Sprintf("--%s=%d", flags.InteropStartIndex.Name, offset),
		)
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
	v.cmd = cmd

	return cmd.Wait()
}

// Started checks whether validator node is started and ready to be queried.
func (v *ValidatorNode) Started() <-chan struct{} {
	return v.started
}

// Pause pauses the component and its underlying process.
func (v *ValidatorNode) Pause() error {
	return v.cmd.Process.Signal(syscall.SIGSTOP)
}

// Resume resumes the component and its underlying process.
func (v *ValidatorNode) Resume() error {
	return v.cmd.Process.Signal(syscall.SIGCONT)
}

// Stop stops the component and its underlying process.
func (v *ValidatorNode) Stop() error {
	return v.cmd.Process.Kill()
}

// SendAndMineDeposits sends the requested amount of deposits and mines the chain after to ensure the deposits are seen.
func SendAndMineDeposits(keystorePath string, validatorNum, offset int, partial bool) error {
	client, err := rpc.DialHTTP(fmt.Sprintf("http://127.0.0.1:%d", e2e.TestParams.Ports.Eth1RPCPort))
	if err != nil {
		return err
	}
	defer client.Close()
	web3 := ethclient.NewClient(client)

	keystoreBytes, err := os.ReadFile(keystorePath) // #nosec G304
	if err != nil {
		return err
	}
	if err = sendDeposits(web3, keystoreBytes, validatorNum, offset, partial); err != nil {
		return err
	}
	mineKey, err := keystore.DecryptKey(keystoreBytes, eth1.KeystorePassword)
	if err != nil {
		return err
	}
	if err = eth1.WaitForBlocks(web3, mineKey, params.BeaconConfig().Eth1FollowDistance); err != nil {
		return fmt.Errorf("failed to mine blocks %w", err)
	}
	return nil
}

// sendDeposits uses the passed in web3 and keystore bytes to send the requested deposits.
func sendDeposits(web3 *ethclient.Client, keystoreBytes []byte, num, offset int, partial bool) error {
	txOps, err := bind.NewTransactorWithChainID(bytes.NewReader(keystoreBytes), eth1.KeystorePassword, big.NewInt(eth1.NetworkId))
	if err != nil {
		return err
	}
	txOps.GasLimit = depositGasLimit
	txOps.Context = context.Background()
	nonce, err := web3.PendingNonceAt(context.Background(), txOps.From)
	if err != nil {
		return err
	}
	txOps.Nonce = big.NewInt(0).SetUint64(nonce)

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

func createProposerSettingsPath(pubkeys []string, validatorIndex int) (string, error) {
	testNetDir := e2e.TestParams.TestPath + fmt.Sprintf("/proposer-settings/validator_%d", validatorIndex)
	configPath := filepath.Join(testNetDir, "config.json")
	if len(pubkeys) == 0 {
		return "", errors.New("number of validators must be greater than 0")
	}
	var proposerSettingsPayload validator_service_config.ProposerSettingsPayload
	if len(pubkeys) == 1 {
		proposerSettingsPayload = validator_service_config.ProposerSettingsPayload{
			DefaultConfig: &validator_service_config.ProposerOptionPayload{
				FeeRecipient: DefaultFeeRecipientAddress,
			},
		}
	} else {
		config := make(map[string]*validator_service_config.ProposerOptionPayload)

		for i, pubkey := range pubkeys {
			// Create an account
			byteval, err := hexutil.Decode(pubkey)
			if err != nil {
				return "", err
			}
			deterministicFeeRecipient := common.HexToAddress(hexutil.Encode(byteval[:fieldparams.FeeRecipientLength])).Hex()
			config[pubkeys[i]] = &validator_service_config.ProposerOptionPayload{
				FeeRecipient: deterministicFeeRecipient,
			}
		}
		proposerSettingsPayload = validator_service_config.ProposerSettingsPayload{
			ProposerConfig: config,
			DefaultConfig: &validator_service_config.ProposerOptionPayload{
				FeeRecipient: DefaultFeeRecipientAddress,
			},
		}
	}
	jsonBytes, err := json.Marshal(proposerSettingsPayload)
	if err != nil {
		return "", err
	}
	if err := file.MkdirAll(testNetDir); err != nil {
		return "", err
	}
	if err := file.WriteFile(configPath, jsonBytes); err != nil {
		return "", err
	}
	return configPath, nil
}
