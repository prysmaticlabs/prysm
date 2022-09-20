package endtoend

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/components"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/components/eth1"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/v3/testing/endtoend/params"
	e2etypes "github.com/prysmaticlabs/prysm/v3/testing/endtoend/types"
	"golang.org/x/sync/errgroup"
)

type componentHandler struct {
	t                        *testing.T
	cfg                      *e2etypes.E2EConfig
	ctx                      context.Context
	done                     func()
	group                    *errgroup.Group
	keygen                   e2etypes.ComponentRunner
	tracingSink              e2etypes.ComponentRunner
	web3Signer               e2etypes.ComponentRunner
	bootnode                 e2etypes.ComponentRunner
	eth1Miner                e2etypes.ComponentRunner
	eth1Proxy                e2etypes.MultipleComponentRunners
	eth1Nodes                e2etypes.MultipleComponentRunners
	beaconNodes              e2etypes.MultipleComponentRunners
	validatorNodes           e2etypes.MultipleComponentRunners
	lighthouseBeaconNodes    e2etypes.MultipleComponentRunners
	lighthouseValidatorNodes e2etypes.MultipleComponentRunners
}

func NewComponentHandler(cfg *e2etypes.E2EConfig, t *testing.T) *componentHandler {
	return &componentHandler{cfg: cfg, t: t}
}

func (c *componentHandler) setup() {
	t, config := c.t, c.cfg
	t.Logf("Shard index: %d\n", e2e.TestParams.TestShardIndex)
	t.Logf("Starting time: %s\n", time.Now().String())
	t.Logf("Log Path: %s\n", e2e.TestParams.LogPath)

	minGenesisActiveCount := int(params.BeaconConfig().MinGenesisActiveValidatorCount)
	multiClientActive := e2e.TestParams.LighthouseBeaconNodeCount > 0
	var keyGen e2etypes.ComponentRunner
	var lighthouseValidatorNodes e2etypes.MultipleComponentRunners
	var lighthouseNodes *components.LighthouseBeaconNodeSet

	c.ctx, c.done = context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(c.ctx)

	tracingSink := components.NewTracingSink(config.TracingSinkEndpoint)
	g.Go(func() error {
		return tracingSink.Start(ctx)
	})
	c.tracingSink = tracingSink

	if multiClientActive {
		keyGen = components.NewKeystoreGenerator()

		// Generate lighthouse keystores.
		g.Go(func() error {
			return keyGen.Start(ctx)
		})
		c.keygen = keyGen
	}

	var web3RemoteSigner *components.Web3RemoteSigner
	if config.UseWeb3RemoteSigner {
		web3RemoteSigner = components.NewWeb3RemoteSigner()
		g.Go(func() error {
			if err := web3RemoteSigner.Start(ctx); err != nil {
				return errors.Wrap(err, "failed to start web3 remote signer")
			}
			return nil
		})
		c.web3Signer = web3RemoteSigner

	}

	// Boot node.
	bootNode := components.NewBootNode()
	g.Go(func() error {
		if err := bootNode.Start(ctx); err != nil {
			return errors.Wrap(err, "failed to start bootnode")
		}
		return nil
	})
	c.bootnode = bootNode

	// ETH1 miner.
	eth1Miner := eth1.NewMiner()
	g.Go(func() error {
		if err := helpers.ComponentsStarted(ctx, []e2etypes.ComponentRunner{bootNode}); err != nil {
			return errors.Wrap(err, "sending and mining deposits require ETH1 nodes to run")
		}
		eth1Miner.SetBootstrapENR(bootNode.ENR())
		if err := eth1Miner.Start(ctx); err != nil {
			return errors.Wrap(err, "failed to start the ETH1 miner")
		}
		return nil
	})
	c.eth1Miner = eth1Miner

	// ETH1 non-mining nodes.
	eth1Nodes := eth1.NewNodeSet()
	g.Go(func() error {
		if err := helpers.ComponentsStarted(ctx, []e2etypes.ComponentRunner{eth1Miner}); err != nil {
			return errors.Wrap(err, "sending and mining deposits require ETH1 nodes to run")
		}
		eth1Nodes.SetMinerENR(eth1Miner.ENR())
		if err := eth1Nodes.Start(ctx); err != nil {
			return errors.Wrap(err, "failed to start ETH1 nodes")
		}
		return nil
	})
	c.eth1Nodes = eth1Nodes

	g.Go(func() error {
		if err := helpers.ComponentsStarted(ctx, []e2etypes.ComponentRunner{eth1Nodes}); err != nil {
			return errors.Wrap(err, "sending and mining deposits require ETH1 nodes to run")
		}
		if err := components.SendAndMineDeposits(eth1Miner.KeystorePath(), minGenesisActiveCount, 0, true /* partial */); err != nil {
			return errors.Wrap(err, "failed to send and mine deposits")
		}
		return nil
	})

	if config.TestCheckpointSync {
		appendDebugEndpoints(config)
	}
	// Proxies
	proxies := eth1.NewProxySet()
	g.Go(func() error {
		if err := helpers.ComponentsStarted(ctx, []e2etypes.ComponentRunner{eth1Nodes}); err != nil {
			return errors.Wrap(err, "beacon nodes require ETH1 and boot node to run")
		}
		if err := proxies.Start(ctx); err != nil {
			return errors.Wrap(err, "failed to start proxies")
		}
		return nil
	})
	c.eth1Proxy = proxies

	// Beacon nodes.
	beaconNodes := components.NewBeaconNodes(config)
	g.Go(func() error {
		if err := helpers.ComponentsStarted(ctx, []e2etypes.ComponentRunner{eth1Nodes, proxies, bootNode}); err != nil {
			return errors.Wrap(err, "beacon nodes require ETH1 and boot node to run")
		}
		beaconNodes.SetENR(bootNode.ENR())
		if err := beaconNodes.Start(ctx); err != nil {
			return errors.Wrap(err, "failed to start beacon nodes")
		}
		return nil
	})
	c.beaconNodes = beaconNodes

	if multiClientActive {
		lighthouseNodes = components.NewLighthouseBeaconNodes(config)
		g.Go(func() error {
			if err := helpers.ComponentsStarted(ctx, []e2etypes.ComponentRunner{eth1Nodes, proxies, bootNode, beaconNodes}); err != nil {
				return errors.Wrap(err, "lighthouse beacon nodes require ETH1 and boot node to run")
			}
			lighthouseNodes.SetENR(bootNode.ENR())
			if err := lighthouseNodes.Start(ctx); err != nil {
				return errors.Wrap(err, "failed to start lighthouse beacon nodes")
			}
			return nil
		})
		c.lighthouseBeaconNodes = lighthouseNodes
	}
	// Validator nodes.
	validatorNodes := components.NewValidatorNodeSet(config)
	g.Go(func() error {
		comps := []e2etypes.ComponentRunner{beaconNodes}
		if config.UseWeb3RemoteSigner {
			comps = append(comps, web3RemoteSigner)
		}
		if err := helpers.ComponentsStarted(ctx, comps); err != nil {
			return errors.Wrap(err, "validator nodes require components to run")
		}
		if err := validatorNodes.Start(ctx); err != nil {
			return errors.Wrap(err, "failed to start validator nodes")
		}
		return nil
	})
	c.validatorNodes = validatorNodes

	if multiClientActive {
		// Lighthouse Validator nodes.
		lighthouseValidatorNodes = components.NewLighthouseValidatorNodeSet(config)
		g.Go(func() error {
			if err := helpers.ComponentsStarted(ctx, []e2etypes.ComponentRunner{keyGen, lighthouseNodes}); err != nil {
				return errors.Wrap(err, "validator nodes require beacon nodes to run")
			}
			if err := lighthouseValidatorNodes.Start(ctx); err != nil {
				return errors.Wrap(err, "failed to start validator nodes")
			}
			return nil
		})
		c.lighthouseValidatorNodes = lighthouseValidatorNodes
	}
	c.group = g
}

func (c *componentHandler) required() []e2etypes.ComponentRunner {
	multiClientActive := e2e.TestParams.LighthouseBeaconNodeCount > 0
	requiredComponents := []e2etypes.ComponentRunner{
		c.tracingSink, c.eth1Nodes, c.bootnode, c.beaconNodes, c.validatorNodes, c.eth1Proxy,
	}
	if multiClientActive {
		requiredComponents = append(requiredComponents, []e2etypes.ComponentRunner{c.keygen, c.lighthouseBeaconNodes, c.lighthouseValidatorNodes}...)
	}
	return requiredComponents
}

func appendDebugEndpoints(cfg *e2etypes.E2EConfig) {
	debug := []string{
		"--enable-debug-rpc-endpoints",
		"--grpc-max-msg-size=65568081",
	}
	cfg.BeaconFlags = append(cfg.BeaconFlags, debug...)
}
