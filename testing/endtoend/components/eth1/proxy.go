package eth1

import (
	"context"
	"fmt"
	"os"
	"path"
	"strconv"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/testing/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/testing/endtoend/params"
	e2etypes "github.com/prysmaticlabs/prysm/testing/endtoend/types"
	proxy "github.com/prysmaticlabs/prysm/testing/middleware/engine-api-proxy"
	log "github.com/sirupsen/logrus"
)

// ProxySet represents a set of proxies for the engine-api.
type ProxySet struct {
	e2etypes.ComponentRunner
	started chan struct{}
	proxies []e2etypes.ComponentRunner
}

// NewProxySet creates and returns a set of engine-api proxies.
func NewProxySet() *ProxySet {
	return &ProxySet{
		started: make(chan struct{}, 1),
	}
}

// Start starts all the proxies in set.
func (s *ProxySet) Start(ctx context.Context) error {

	totalNodeCount := e2e.TestParams.BeaconNodeCount + e2e.TestParams.LighthouseBeaconNodeCount
	nodes := make([]e2etypes.ComponentRunner, totalNodeCount)
	for i := 0; i < totalNodeCount; i++ {
		// We start indexing nodes from 1 because the miner has an implicit 0 index.
		nodes[i] = NewProxy(i)
	}
	s.proxies = nodes

	// Wait for all nodes to finish their job (blocking).
	// Once nodes are ready passed in handler function will be called.
	return helpers.WaitOnNodes(ctx, nodes, func() {
		// All nodes started, close channel, so that all services waiting on a set, can proceed.
		close(s.started)
	})
}

// Started checks whether proxy set is started and all proxies are ready to be queried.
func (s *ProxySet) Started() <-chan struct{} {
	return s.started
}

// Pause pauses the component and its underlying process.
func (s *ProxySet) Pause() error {
	for _, n := range s.proxies {
		if err := n.Pause(); err != nil {
			return err
		}
	}
	return nil
}

// Resume resumes the component and its underlying process.
func (s *ProxySet) Resume() error {
	for _, n := range s.proxies {
		if err := n.Resume(); err != nil {
			return err
		}
	}
	return nil
}

// Stop stops the component and its underlying process.
func (s *ProxySet) Stop() error {
	for _, n := range s.proxies {
		if err := n.Stop(); err != nil {
			return err
		}
	}
	return nil
}

// PauseAtIndex pauses the component and its underlying process at the desired index.
func (s *ProxySet) PauseAtIndex(i int) error {
	if i >= len(s.proxies) {
		return errors.Errorf("provided index exceeds slice size: %d >= %d", i, len(s.proxies))
	}
	return s.proxies[i].Pause()
}

// ResumeAtIndex resumes the component and its underlying process at the desired index.
func (s *ProxySet) ResumeAtIndex(i int) error {
	if i >= len(s.proxies) {
		return errors.Errorf("provided index exceeds slice size: %d >= %d", i, len(s.proxies))
	}
	return s.proxies[i].Resume()
}

// StopAtIndex stops the component and its underlying process at the desired index.
func (s *ProxySet) StopAtIndex(i int) error {
	if i >= len(s.proxies) {
		return errors.Errorf("provided index exceeds slice size: %d >= %d", i, len(s.proxies))
	}
	return s.proxies[i].Stop()
}

// Proxy represents an engine-api proxy.
type Proxy struct {
	e2etypes.ComponentRunner
	started chan struct{}
	index   int
	cancel  func()
}

// NewProxy creates and returns an engine-api proxy.
func NewProxy(index int) *Proxy {
	return &Proxy{
		started: make(chan struct{}, 1),
		index:   index,
	}
}

// Start runs a proxy.
func (node *Proxy) Start(ctx context.Context) error {
	file, err := os.Create(path.Join(e2e.TestParams.LogPath, "eth1_proxy_"+strconv.Itoa(node.index)+".log"))
	if err != nil {
		return err
	}
	opts := []proxy.Option{
		proxy.WithDestinationAddress(fmt.Sprintf("http://127.0.0.1:%d", e2e.TestParams.Ports.Eth1AuthRPCPort+node.index)),
		proxy.WithPort(e2e.TestParams.Ports.Eth1ProxyPort + node.index),
		proxy.WithLogger(log.New()),
		proxy.WithLogFile(file),
	}
	nProxy, err := proxy.New(opts...)
	if err != nil {
		return err
	}
	log.Infof("Starting eth1 proxy %d with port: %d and file %s", node.index, e2e.TestParams.Ports.Eth1ProxyPort+node.index, file.Name())

	// Set cancel into context.
	ctx, cancel := context.WithCancel(ctx)
	node.cancel = cancel
	// Mark node as ready.
	close(node.started)
	return nProxy.Start(ctx)
}

// Started checks whether the eth1 proxy is started and ready to be queried.
func (node *Proxy) Started() <-chan struct{} {
	return node.started
}

// Pause pauses the component and its underlying process.
func (node *Proxy) Pause() error {
	// no-op
	return nil
}

// Resume resumes the component and its underlying process.
func (node *Proxy) Resume() error {
	// no-op
	return nil
}

// Stop kills the component and its underlying process.
func (node *Proxy) Stop() error {
	node.cancel()
	return nil
}
