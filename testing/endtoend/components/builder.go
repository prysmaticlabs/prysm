package components

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/io/file"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/v3/testing/endtoend/params"
	e2etypes "github.com/prysmaticlabs/prysm/v3/testing/endtoend/types"
)

// BuilderSet represents a set of builders for the validators running via a relay.
type BuilderSet struct {
	e2etypes.ComponentRunner
	started  chan struct{}
	builders []e2etypes.ComponentRunner
}

// NewBuilderSet creates and returns a set of builders.
func NewBuilderSet() *BuilderSet {
	return &BuilderSet{
		started: make(chan struct{}, 1),
	}
}

// Start starts all the builders in set.
func (s *BuilderSet) Start(ctx context.Context) error {
	totalNodeCount := 1
	nodes := make([]e2etypes.ComponentRunner, totalNodeCount)
	for i := 0; i < totalNodeCount; i++ {
		nodes[i] = NewBuilder(i)
	}
	s.builders = nodes

	// Wait for all nodes to finish their job (blocking).
	// Once nodes are ready passed in handler function will be called.
	return helpers.WaitOnNodes(ctx, nodes, func() {
		// All nodes started, close channel, so that all services waiting on a set, can proceed.
		close(s.started)
	})
}

// Started checks whether builder set is started and all builders are ready to be queried.
func (s *BuilderSet) Started() <-chan struct{} {
	return s.started
}

// Pause pauses the component and its underlying process.
func (s *BuilderSet) Pause() error {
	for _, n := range s.builders {
		if err := n.Pause(); err != nil {
			return err
		}
	}
	return nil
}

// Resume resumes the component and its underlying process.
func (s *BuilderSet) Resume() error {
	for _, n := range s.builders {
		if err := n.Resume(); err != nil {
			return err
		}
	}
	return nil
}

// Stop stops the component and its underlying process.
func (s *BuilderSet) Stop() error {
	for _, n := range s.builders {
		if err := n.Stop(); err != nil {
			return err
		}
	}
	return nil
}

// PauseAtIndex pauses the component and its underlying process at the desired index.
func (s *BuilderSet) PauseAtIndex(i int) error {
	if i >= len(s.builders) {
		return errors.Errorf("provided index exceeds slice size: %d >= %d", i, len(s.builders))
	}
	return s.builders[i].Pause()
}

// ResumeAtIndex resumes the component and its underlying process at the desired index.
func (s *BuilderSet) ResumeAtIndex(i int) error {
	if i >= len(s.builders) {
		return errors.Errorf("provided index exceeds slice size: %d >= %d", i, len(s.builders))
	}
	return s.builders[i].Resume()
}

// StopAtIndex stops the component and its underlying process at the desired index.
func (s *BuilderSet) StopAtIndex(i int) error {
	if i >= len(s.builders) {
		return errors.Errorf("provided index exceeds slice size: %d >= %d", i, len(s.builders))
	}
	return s.builders[i].Stop()
}

// ComponentAtIndex returns the component at the provided index.
func (s *BuilderSet) ComponentAtIndex(i int) (e2etypes.ComponentRunner, error) {
	if i >= len(s.builders) {
		return nil, errors.Errorf("provided index exceeds slice size: %d >= %d", i, len(s.builders))
	}
	return s.builders[i], nil
}

// Builder represents a block builder.
type Builder struct {
	e2etypes.ComponentRunner
	started     chan struct{}
	index       int
	engineProxy *exec.Cmd
	cancel      func()
}

// NewBuilder creates and returns a builder.
func NewBuilder(index int) *Builder {
	return &Builder{
		started: make(chan struct{}, 1),
		index:   index,
	}
}

// Start runs a builder.
func (node *Builder) Start(ctx context.Context) error {
	binaryPath, found := bazel.FindBinary("external/mev_rs", "mev")
	if !found {
		log.Info(binaryPath)
		log.Error("mev rs binary not found")
	}

	cfgPath, err := node.saveToml()
	if err != nil {
		return err
	}
	args := []string{
		"build",
		"mempool",
		cfgPath,
	}
	cmd := exec.CommandContext(ctx, binaryPath, args...) // #nosec G204 -- Safe
	// Write stderr to log files.
	stderr, err := os.Create(path.Join(e2e.TestParams.LogPath, fmt.Sprintf("builder_%d_stderr.log", node.index)))
	if err != nil {
		return err
	}
	defer func() {
		if err := stderr.Close(); err != nil {
			log.WithError(err).Error("Failed to close stderr file")
		}
	}()
	cmd.Stderr = stderr
	log.Infof("Starting builder %d with flags: %s %s", node.index, strings.Join(args, " "), binaryPath)
	if err = cmd.Start(); err != nil {
		return fmt.Errorf("failed to start builder: %w", err)
	}

	// Mark node as ready.
	close(node.started)

	node.engineProxy = cmd
	return cmd.Wait()
}

// Started checks whether the builder is started and ready to be queried.
func (node *Builder) Started() <-chan struct{} {
	return node.started
}

// Pause pauses the component and its underlying process.
func (node *Builder) Pause() error {
	// no-op
	return nil
}

// Resume resumes the component and its underlying process.
func (node *Builder) Resume() error {
	// no-op
	return nil
}

// Stop kills the component and its underlying process.
func (node *Builder) Stop() error {
	node.cancel()
	return nil
}

func (node *Builder) saveToml() (string, error) {
	builderToml := fmt.Sprintf(`[builder]
host = "127.0.0.1"
port = 28546
beacon_api_endpoint = "http://127.0.0.1:%d"

[builder.engine_api_proxy]
host = "127.0.0.1"
port = %d
engine_api_endpoint = "http://127.0.0.1:%d"`, e2e.TestParams.Ports.PrysmBeaconNodeGatewayPort+node.index,
		e2e.TestParams.Ports.Eth1ProxyPort+node.index, e2e.TestParams.Ports.Eth1AuthRPCPort+node.index)
	log.Infof("toml %s", builderToml)
	builderPath := path.Join(e2e.TestParams.TestPath, "builder/"+strconv.Itoa(node.index)+"/")
	cfgPath := path.Join(builderPath, "config.toml")
	if err := file.MkdirAll(builderPath); err != nil {
		return "", err
	}
	if err := file.WriteFile(cfgPath, []byte(builderToml)); err != nil {
		return "", err
	}
	return cfgPath, nil
}

func parseJWTSecretFromFile(jwtSecretFile string) ([]byte, error) {
	enc, err := file.ReadFileAsBytes(jwtSecretFile)
	if err != nil {
		return nil, err
	}
	strData := strings.TrimSpace(string(enc))
	if len(strData) == 0 {
		return nil, fmt.Errorf("provided JWT secret in file %s cannot be empty", jwtSecretFile)
	}
	secret, err := hex.DecodeString(strings.TrimPrefix(strData, "0x"))
	if err != nil {
		return nil, err
	}
	if len(secret) < 32 {
		return nil, errors.New("provided JWT secret should be a hex string of at least 32 bytes")
	}
	return secret, nil
}
