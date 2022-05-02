package components

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/io/file"
	"github.com/prysmaticlabs/prysm/runtime/interop"
	"github.com/prysmaticlabs/prysm/testing/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/testing/endtoend/params"
	e2etypes "github.com/prysmaticlabs/prysm/testing/endtoend/types"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

var _ e2etypes.ComponentRunner = (*LighthouseValidatorNode)(nil)
var _ e2etypes.ComponentRunner = (*LighthouseValidatorNodeSet)(nil)

// LighthouseValidatorNodeSet represents set of lighthouse validator nodes.
type LighthouseValidatorNodeSet struct {
	e2etypes.ComponentRunner
	config  *e2etypes.E2EConfig
	started chan struct{}
}

// NewLighthouseValidatorNodeSet creates and returns a set of lighthouse validator nodes.
func NewLighthouseValidatorNodeSet(config *e2etypes.E2EConfig) *LighthouseValidatorNodeSet {
	return &LighthouseValidatorNodeSet{
		config:  config,
		started: make(chan struct{}, 1),
	}
}

// Start starts the configured amount of validators, also sending and mining their deposits.
func (s *LighthouseValidatorNodeSet) Start(ctx context.Context) error {
	// Always using genesis count since using anything else would be difficult to test for.
	validatorNum := int(params.BeaconConfig().MinGenesisActiveValidatorCount)
	lighthouseBeaconNum := e2e.TestParams.LighthouseBeaconNodeCount
	prysmBeaconNum := e2e.TestParams.BeaconNodeCount
	beaconNodeNum := lighthouseBeaconNum + prysmBeaconNum
	if validatorNum%beaconNodeNum != 0 {
		return errors.New("validator count is not easily divisible by beacon node count")
	}
	validatorsPerNode := validatorNum / beaconNodeNum

	// Create validator nodes.
	nodes := make([]e2etypes.ComponentRunner, lighthouseBeaconNum)
	for i := 0; i < lighthouseBeaconNum; i++ {
		offsetIdx := i + prysmBeaconNum
		nodes[i] = NewLighthouseValidatorNode(s.config, validatorsPerNode, i, validatorsPerNode*offsetIdx)
	}

	// Wait for all nodes to finish their job (blocking).
	// Once nodes are ready passed in handler function will be called.
	return helpers.WaitOnNodes(ctx, nodes, func() {
		// All nodes stated, close channel, so that all services waiting on a set, can proceed.
		close(s.started)
	})
}

// Started checks whether validator node set is started and all nodes are ready to be queried.
func (s *LighthouseValidatorNodeSet) Started() <-chan struct{} {
	return s.started
}

// LighthouseValidatorNode represents a lighthouse validator node.
type LighthouseValidatorNode struct {
	e2etypes.ComponentRunner
	config       *e2etypes.E2EConfig
	started      chan struct{}
	validatorNum int
	index        int
	offset       int
}

// NewLighthouseValidatorNode creates and returns a lighthouse validator node.
func NewLighthouseValidatorNode(config *e2etypes.E2EConfig, validatorNum, index, offset int) *LighthouseValidatorNode {
	return &LighthouseValidatorNode{
		config:       config,
		validatorNum: validatorNum,
		index:        index,
		offset:       offset,
		started:      make(chan struct{}, 1),
	}
}

// Start starts a validator client.
func (v *LighthouseValidatorNode) Start(ctx context.Context) error {
	binaryPath, found := bazel.FindBinary("external/lighthouse", "lighthouse")
	if !found {
		log.Info(binaryPath)
		log.Error("validator binary not found")
	}

	_, _, index, _ := v.config, v.validatorNum, v.index, v.offset
	beaconRPCPort := e2e.TestParams.Ports.PrysmBeaconNodeRPCPort + index
	if beaconRPCPort >= e2e.TestParams.Ports.PrysmBeaconNodeRPCPort+e2e.TestParams.BeaconNodeCount {
		// Point any extra validator clients to a node we know is running.
		beaconRPCPort = e2e.TestParams.Ports.PrysmBeaconNodeRPCPort
	}
	kPath := e2e.TestParams.TestPath + fmt.Sprintf("/lighthouse-validator-%d", index)
	testNetDir := e2e.TestParams.TestPath + fmt.Sprintf("/lighthouse-testnet-%d", index)
	httpPort := e2e.TestParams.Ports.LighthouseBeaconNodeHTTPPort
	// In the event we want to run a LH validator with a non LH
	// beacon node, we split half the validators to run with
	// lighthouse and the other half with prysm.
	if v.config.UseValidatorCrossClient && index%2 == 0 {
		httpPort = e2e.TestParams.Ports.PrysmBeaconNodeGatewayPort
	}
	args := []string{
		"validator_client",
		"--debug-level=debug",
		"--init-slashing-protection",
		fmt.Sprintf("--datadir=%s", kPath),
		fmt.Sprintf("--testnet-dir=%s", testNetDir),
		fmt.Sprintf("--beacon-nodes=http://localhost:%d", httpPort+index),
	}

	cmd := exec.CommandContext(ctx, binaryPath, args...) // #nosec G204 -- Safe

	// Write stdout and stderr to log files.
	stdout, err := os.Create(path.Join(e2e.TestParams.LogPath, fmt.Sprintf("lighthouse_validator_%d_stdout.log", index)))
	if err != nil {
		return err
	}
	stderr, err := os.Create(path.Join(e2e.TestParams.LogPath, fmt.Sprintf("lighthouse_validator_%d_stderr.log", index)))
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

	log.Infof("Starting lighthouse validator client %d with flags: %s %s", index, binaryPath, strings.Join(args, " "))
	if err = cmd.Start(); err != nil {
		return err
	}

	// Mark node as ready.
	close(v.started)

	return cmd.Wait()
}

// Started checks whether validator node is started and ready to be queried.
func (v *LighthouseValidatorNode) Started() <-chan struct{} {
	return v.started
}

type KeystoreGenerator struct {
	started chan struct{}
}

func NewKeystoreGenerator() *KeystoreGenerator {
	return &KeystoreGenerator{started: make(chan struct{})}
}

func (k *KeystoreGenerator) Start(ctx context.Context) error {
	validatorNum := int(params.BeaconConfig().MinGenesisActiveValidatorCount)
	lighthouseBeaconNum := e2e.TestParams.LighthouseBeaconNodeCount
	prysmBeaconNum := e2e.TestParams.BeaconNodeCount
	beaconNodeNum := lighthouseBeaconNum + prysmBeaconNum
	if validatorNum%beaconNodeNum != 0 {
		return errors.New("validator count is not easily divisible by beacon node count")
	}
	validatorsPerNode := validatorNum / beaconNodeNum

	for i := 0; i < lighthouseBeaconNum; i++ {
		offsetIdx := i + prysmBeaconNum
		_, err := setupKeystores(i, validatorsPerNode*offsetIdx, validatorsPerNode)
		if err != nil {
			return err
		}
		log.Infof("Generated lighthouse keystores from %d onwards with %d keys", validatorsPerNode*offsetIdx, validatorsPerNode)
	}
	// Mark component as ready.
	close(k.started)
	return nil
}

func (k *KeystoreGenerator) Started() <-chan struct{} {
	return k.started
}

func setupKeystores(valClientIdx, startIdx, numOfKeys int) (string, error) {
	testNetDir := e2e.TestParams.TestPath + fmt.Sprintf("/lighthouse-validator-%d", valClientIdx)
	if err := file.MkdirAll(testNetDir); err != nil {
		return "", err
	}
	secretsPath := filepath.Join(testNetDir, "secrets")
	validatorKeystorePath := filepath.Join(testNetDir, "validators")
	if err := file.MkdirAll(secretsPath); err != nil {
		return "", err
	}
	if err := file.MkdirAll(validatorKeystorePath); err != nil {
		return "", err
	}
	privKeys, pubKeys, err := interop.DeterministicallyGenerateKeys(uint64(startIdx), uint64(numOfKeys))
	if err != nil {
		return "", err
	}
	encryptor := keystorev4.New()
	// Use lighthouse's default password for their insecure keystores.
	password := "222222222222222222222222222222222222222222222222222"
	for i, pk := range pubKeys {
		pubKeyBytes := pk.Marshal()
		cryptoFields, err := encryptor.Encrypt(privKeys[i].Marshal(), password)
		if err != nil {
			return "", errors.Wrapf(
				err,
				"could not encrypt secret key for public key %#x",
				pubKeyBytes,
			)
		}
		id, err := uuid.NewRandom()
		if err != nil {
			return "", err
		}
		kStore := &keymanager.Keystore{
			Crypto:  cryptoFields,
			ID:      id.String(),
			Pubkey:  fmt.Sprintf("%x", pubKeyBytes),
			Version: encryptor.Version(),
			Name:    encryptor.Name(),
		}

		fPath := filepath.Join(secretsPath, "0x"+kStore.Pubkey)
		if err := file.WriteFile(fPath, []byte(password)); err != nil {
			return "", err
		}
		keystorePath := filepath.Join(validatorKeystorePath, "0x"+kStore.Pubkey)
		if err := file.MkdirAll(keystorePath); err != nil {
			return "", err
		}
		fPath = filepath.Join(keystorePath, "voting-keystore.json")
		encodedFile, err := json.MarshalIndent(kStore, "", "\t")
		if err != nil {
			return "", errors.Wrap(err, "could not marshal keystore to JSON file")
		}
		if err := file.WriteFile(fPath, encodedFile); err != nil {
			return "", err
		}
	}
	return testNetDir, nil
}
