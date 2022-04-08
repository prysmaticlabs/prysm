// Package endtoend performs full a end-to-end test for Prysm,
// including spinning up an ETH1 dev chain, sending deposits to the deposit
// contract, and making sure the beacon node and validators are running and
// performing properly for a few epochs.
package endtoend

import (
	"context"
	"fmt"
	"github.com/prysmaticlabs/prysm/io/file"
	"os"
	"path"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/testing/endtoend/e2ez"
	"github.com/prysmaticlabs/prysm/api/client/beacon"
	"github.com/prysmaticlabs/prysm/io/file"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/api/client/beacon"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/proto/eth/service"
	v1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/endtoend/components"
	"github.com/prysmaticlabs/prysm/testing/endtoend/components/eth1"
	ev "github.com/prysmaticlabs/prysm/testing/endtoend/evaluators"
	"github.com/prysmaticlabs/prysm/testing/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/testing/endtoend/params"
	e2etypes "github.com/prysmaticlabs/prysm/testing/endtoend/types"
	"github.com/prysmaticlabs/prysm/testing/require"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	// allNodesStartTimeout defines period after which nodes are considered
	// stalled (safety measure for nodes stuck at startup, shouldn't normally happen).
	allNodesStartTimeout = 5 * time.Minute

	// errGeneralCode is used to represent the string value for all general process errors.
	errGeneralCode = "exit status 1"
)

func init() {
	transition.SkipSlotCache.Disable()
}

// testRunner abstracts E2E test configuration and running.
type testRunner struct {
	t      *testing.T
	config *e2etypes.E2EConfig
	z      *e2ez.Server
	ctx      context.Context
	doneChan context.CancelFunc
	group    *errgroup.Group
}

// newTestRunner creates E2E test runner.
func newTestRunner(t *testing.T, config *e2etypes.E2EConfig) *testRunner {
	ctx, done := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)
	return &testRunner{
		t:      t,
		config: config,
		z:      e2ez.NewServer(),
		ctx:      ctx,
		doneChan: done,
		group:    g,
	}
}

type zPageMenu struct {}

func (z *zPageMenu) ZPath() string {
	return "/"
}

func (z *zPageMenu) ZMarkdown() (string, error) {
	return `
e2e admin
===========

- [prysm beacon nodes](/beacon-nodes)
- [lh beacon nodes](/lh-beacon-nodes)
`, nil
}

func (z *zPageMenu) ZChildren() []e2ez.ZPage {
	return []e2ez.ZPage{}
}

// run executes configured E2E test.
func (r *testRunner) run() {
	t, config := r.t, r.config
	err := config.WriteBeaconChainConfig()
	if err != nil {
		t.Fatalf("failed to write BeaconChainConfig to bazel sandbox")
	}

	t.Logf("Shard index: %d\n", e2e.TestParams.TestShardIndex)
	t.Logf("Starting time: %s\n", time.Now().String())
	t.Logf("Log Path: %s\n", e2e.TestParams.LogPath)

	if e2e.TestParams.ZPageAddr == "" {
		e2e.TestParams.ZPageAddr = ":8080"
	}

	// we need debug turned on and max ssz payload bumped up when running checkpoint sync teste2e.TestParams.BeaconNodeCounts
	if config.TestSync {
		config.BeaconFlags = appendDebugEndpoints(config.BeaconFlags)
	}
	minGenesisActiveCount := int(params.BeaconConfig().MinGenesisActiveValidatorCount)
	multiClientActive := e2e.TestParams.LighthouseBeaconNodeCount > 0
	var keyGen, lighthouseValidatorNodes e2etypes.ComponentRunner
	var lighthouseNodes *components.LighthouseBeaconNodeSet

	ctx := r.ctx
	g := r.group
	done := r.doneChan

	tracingSink := components.NewTracingSink(config.TracingSinkEndpoint)
	g.Go(func() error {
		return tracingSink.Start(ctx)
	})
	zp := e2ez.NewServer()
	zp.HandleZPages(&zPageMenu{})
	go zp.ListenAndServe(ctx, e2e.TestParams.ZPageAddr)

	if multiClientActive {
		keyGen = components.NewKeystoreGenerator()

		// Generate lighthouse keystores.
		g.Go(func() error {
			return keyGen.Start(ctx)
		})
	}

	// Boot node.
	bootNode := components.NewBootNode()
	g.Go(func() error {
		if err := bootNode.Start(ctx); err != nil {
			return errors.Wrap(err, "failed to start bootnode")
		}
		return nil
	})

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
	g.Go(func() error {
		if err := helpers.ComponentsStarted(ctx, []e2etypes.ComponentRunner{eth1Nodes}); err != nil {
			return errors.Wrap(err, "sending and mining deposits require ETH1 nodes to run")
		}
		if err := components.SendAndMineDeposits(eth1Miner.KeystorePath(), minGenesisActiveCount, 0, true /* partial */); err != nil {
			return errors.Wrap(err, "failed to send and mine deposits")
		}
		return nil
	})

	// Web3 remote signer.
	var web3RemoteSigner *components.Web3RemoteSigner
	if config.UseWeb3RemoteSigner {
		web3RemoteSigner = components.NewWeb3RemoteSigner()
		g.Go(func() error {
			if err := web3RemoteSigner.Start(ctx); err != nil {
				return errors.Wrap(err, "failed to start web3 remote signer")
			}
			return nil
		})
	}

	// Beacon nodes.
	if err := helpers.ComponentsStarted(ctx, []e2etypes.ComponentRunner{bootNode}); err != nil {
		t.Fatal(err, errors.Wrap(err, "beacon nodes require ETH1 and boot node to run"))
	}
	beaconNodes := components.NewBeaconNodes(config, bootNode.ENR(), config.BeaconFlags)
	zp.HandleZPages(beaconNodes)
	g.Go(func() error {
		if err := helpers.ComponentsStarted(ctx, []e2etypes.ComponentRunner{eth1Nodes, bootNode}); err != nil {
			t.Fatal(err, errors.Wrap(err, "beacon nodes require ETH1 and boot node to run"))
		}
		if err := beaconNodes.Start(ctx); err != nil {
			return errors.Wrap(err, "failed to start beacon nodes")
		}
		return nil
	})

	if multiClientActive {
		if err := helpers.ComponentsStarted(ctx, []e2etypes.ComponentRunner{eth1Nodes, bootNode, beaconNodes}); err != nil {
			t.Fatal(errors.Wrap(err, "lighthouse beacon nodes require ETH1 and boot node to run"))
		}
		lighthouseNodes = components.NewLighthouseBeaconNodes(config, bootNode.ENR())
		zp.HandleZPages(lighthouseNodes)
		g.Go(func() error {
			if err := lighthouseNodes.Start(ctx); err != nil {
				return errors.Wrap(err, "failed to start lighthouse beacon nodes")
			}
			return nil
		})
	}
	// Validator nodes.
	validatorNodes := components.NewValidatorNodeSet(config)
	g.Go(func() error {
		comps := []e2etypes.ComponentRunner{beaconNodes}
		if config.UseWeb3RemoteSigner {
			comps = append(comps, web3RemoteSigner)
		}
		if err := helpers.ComponentsStarted(ctx, comps); err != nil {
			return errors.Wrap(err, "validator nodes require beacon nodes to run")
		}
		if err := validatorNodes.Start(ctx); err != nil {
			return errors.Wrap(err, "failed to start validator nodes")
		}
		return nil
	})

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
	}

	// Run E2E evaluators and tests.
	g.Go(func() error {
		// When everything is done, cancel parent context (will stop all spawned nodes).
		defer func() {
			log.Info("All E2E evaluations are finished, cleaning up")
			done()
		}()

		// Wait for all required nodes to start.
		requiredComponents := []e2etypes.ComponentRunner{
			tracingSink, eth1Nodes, bootNode, beaconNodes, validatorNodes,
		}
		if multiClientActive {
			requiredComponents = append(requiredComponents, []e2etypes.ComponentRunner{keyGen, lighthouseNodes, lighthouseValidatorNodes}...)
		}
		ctxAllNodesReady, cancel := context.WithTimeout(ctx, allNodesStartTimeout)
		defer cancel()
		if err := helpers.ComponentsStarted(ctxAllNodesReady, requiredComponents); err != nil {
			return errors.Wrap(err, "components take too long to start")
		}

		// Since defer unwraps in LIFO order, parent context will be closed only after logs are written.
		defer helpers.LogOutput(t)
		if config.UsePprof {
			defer func() {
				log.Info("Writing output pprof files")
				for i := 0; i < e2e.TestParams.BeaconNodeCount; i++ {
					assert.NoError(t, helpers.WritePprofFiles(e2e.TestParams.LogPath, i))
				}
			}()
		}

		// Blocking, wait period varies depending on number of validators.
		r.waitForChainStart()

		// Failing early in case chain doesn't start.
		if t.Failed() {
			return errors.New("chain cannot start")
		}

		r.testDepositsAndTx(ctx, g, eth1Miner.KeystorePath(), []e2etypes.ComponentRunner{beaconNodes})

		// Create GRPC connection to beacon nodes.
		conns, closeConns, err := helpers.NewLocalConnections(ctx, e2e.TestParams.BeaconNodeCount)
		require.NoError(t, err, "Cannot create local connections")
		defer closeConns()

		// Calculate genesis time.
		nodeClient := eth.NewNodeClient(conns[0])
		genesis, err := nodeClient.GetGenesis(context.Background(), &emptypb.Empty{})
		require.NoError(t, err)
		tickingStartTime := helpers.EpochTickerStartTime(genesis)

		// Run assigned evaluators.
		if err := r.runEvaluators(conns, tickingStartTime); err != nil {
			return errors.Wrap(err, "one or more evaluators failed")
		}
		if err := r.testBeaconChainSync(ctx, g, conns, tickingStartTime, bootNode.ENR(), eth1Miner.ENR()); err != nil {
			return errors.Wrap(err, "beacon chain sync test failed")
		}
		if err := r.testDoppelGangerProtection(ctx); err != nil {
			return errors.Wrap(err, "doppel ganger protection check failed")
		}

		// If requested, run sync test.
		if config.TestSync {
			httpEndpoints := helpers.BeaconAPIHostnames(e2e.TestParams.BeaconNodeCount)
			index := e2e.TestParams.BeaconNodeCount
			menr := eth1Miner.ENR()
			benr := bootNode.ENR()
			if err := r.testCheckpointSync(index+1, conns, httpEndpoints[0], benr, menr); err != nil {
				return errors.Wrap(err, "checkpoint sync test failed")
			}
		}

		return nil
	})

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		// At the end of the main evaluator goroutine all nodes are killed, no need to fail the test.
		if strings.Contains(err.Error(), "signal: killed") {
			return
		}
		t.Fatalf("E2E test ended in error: %v", err)
	}
}

func appendDebugEndpoints(flags []string) []string {
	debugFlags := []string{
		"--enable-debug-rpc-endpoints",
		"--grpc-max-msg-size=65568081",
	}
	return append(flags, debugFlags...)
}

// waitForChainStart allows to wait up until beacon nodes are started.
func (r *testRunner) waitForChainStart() {
	// Sleep depending on the count of validators, as generating the genesis state could take some time.
	time.Sleep(time.Duration(params.BeaconConfig().GenesisDelay) * time.Second)
	beaconLogFile, err := os.Open(path.Join(e2e.TestParams.LogPath, fmt.Sprintf(e2e.BeaconNodeLogFileName, 0)))
	require.NoError(r.t, err)

	r.t.Run("chain started", func(t *testing.T) {
		require.NoError(t, helpers.WaitForTextInFile(beaconLogFile, "Chain started in sync service"), "Chain did not start")
	})
}

// runEvaluators executes assigned evaluators.
func (r *testRunner) runEvaluators(conns []*grpc.ClientConn, tickingStartTime time.Time) error {
	t, config := r.t, r.config
	secondsPerEpoch := uint64(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().SecondsPerSlot))
	ticker := helpers.NewEpochTicker(tickingStartTime, secondsPerEpoch)
	for currentEpoch := range ticker.C() {
		wg := new(sync.WaitGroup)
		for _, eval := range config.Evaluators {
			// Fix reference to evaluator as it will be running
			// in a separate goroutine.
			evaluator := eval
			// Only run if the policy says so.
			if !evaluator.Policy(types.Epoch(currentEpoch)) {
				continue
			}
			wg.Add(1)
			go t.Run(fmt.Sprintf(evaluator.Name, currentEpoch), func(t *testing.T) {
				err := evaluator.Evaluation(conns...)
				assert.NoError(t, err, "Evaluation failed for epoch %d: %v", currentEpoch, err)
				wg.Done()
			})
		}
		wg.Wait()

		if t.Failed() || currentEpoch >= config.EpochsToRun-1 {
			ticker.Done()
			if t.Failed() {
				return errors.New("test failed")
			}
			break
		}
	}
	return nil
}

// testDepositsAndTx runs tests when config.TestDeposits is enabled.
func (r *testRunner) testDepositsAndTx(ctx context.Context, g *errgroup.Group,
	keystorePath string, requiredNodes []e2etypes.ComponentRunner) {
	minGenesisActiveCount := int(params.BeaconConfig().MinGenesisActiveValidatorCount)

	depositCheckValidator := components.NewValidatorNode(r.config, int(e2e.DepositCount), e2e.TestParams.BeaconNodeCount, minGenesisActiveCount)
	g.Go(func() error {
		if err := helpers.ComponentsStarted(ctx, requiredNodes); err != nil {
			return fmt.Errorf("deposit check validator node requires beacon nodes to run: %w", err)
		}
		go func() {
			if r.config.TestDeposits {
				log.Info("Running deposit tests")
				err := components.SendAndMineDeposits(keystorePath, int(e2e.DepositCount), minGenesisActiveCount, false /* partial */)
				if err != nil {
					r.t.Fatal(err)
				}
			}
			r.testTxGeneration(ctx, g, keystorePath, []e2etypes.ComponentRunner{})
		}()
		if r.config.TestDeposits {
			return depositCheckValidator.Start(ctx)
		}
		return nil
	})
}

func (r *testRunner) testTxGeneration(ctx context.Context, g *errgroup.Group, keystorePath string, requiredNodes []e2etypes.ComponentRunner) {
	txGenerator := eth1.NewTransactionGenerator(keystorePath, r.config.Seed)
	g.Go(func() error {
		if err := helpers.ComponentsStarted(ctx, requiredNodes); err != nil {
			return fmt.Errorf("transaction generator requires eth1 nodes to be run: %w", err)
		}
		return txGenerator.Start(ctx)
	})
}

func (r *testRunner) waitForMatchingHead(ctx context.Context, check, ref *grpc.ClientConn) error {
	// sleep hack copied from testBeaconChainSync
	// Sleep a second for every 4 blocks that need to be synced for the newly started node.
	secondsPerEpoch := uint64(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().SecondsPerSlot))
	extraSecondsToSync := (r.config.EpochsToRun)*secondsPerEpoch + uint64(params.BeaconConfig().SlotsPerEpoch.Div(4).Mul(r.config.EpochsToRun))
	ctx, cancel := context.WithDeadline(r.ctx, time.Now().Add(time.Second*time.Duration(extraSecondsToSync)))
	pause := time.After(time.Second * 1)
	checkClient := service.NewBeaconChainClient(check)
	refClient := service.NewBeaconChainClient(ref)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			// deadline ensures that the test eventually exits when beacon node fails to sync in a resonable timeframe
			return fmt.Errorf("deadline exceeded waiting for known good block to appear in checkpoint-synced node")
		case <-pause:
			cResp, err := checkClient.GetBlockRoot(ctx, &v1.BlockRequest{BlockId: []byte("head")})
			if err != nil {
				errStatus, ok := status.FromError(err)
				// in the happy path we expect NotFound results until the node has synced
				if ok && errStatus.Code() == codes.NotFound {
					continue
				}
				return fmt.Errorf("error requesting head from 'check' beacon node")
			}
			rResp, err := refClient.GetBlockRoot(ctx, &v1.BlockRequest{BlockId: []byte("head")})
			if err != nil {
				return errors.Wrap(err, "unexpected error requesting head block root from 'ref' beacon node")
			}
			if bytesutil.ToBytes32(cResp.Data.Root) == bytesutil.ToBytes32(rResp.Data.Root) {
				return nil
			}
		}
	}
}

func (r *testRunner) testCheckpointSync(i int, conns []*grpc.ClientConn, bnAPI, enr, minerEnr string) error {
	ethNode := eth1.NewNode(i, minerEnr)
	r.group.Go(func() error {
		return ethNode.Start(r.ctx)
	})
	if err := helpers.ComponentsStarted(r.ctx, []e2etypes.ComponentRunner{ethNode}); err != nil {
		return fmt.Errorf("sync beacon node not ready: %w", err)
	}

	client, err := beacon.NewClient(bnAPI)
	if err != nil {
		return err
	}

	od, err := beacon.DownloadOriginData(r.ctx, client)
	if err != nil {
		return err
	}
	blockPath, err := od.SaveBlock(e2e.TestParams.TestPath)
	if err != nil {
		return err
	}
	statePath, err := od.SaveState(e2e.TestParams.TestPath)
	if err != nil {
		return err
	}
	gb, err := client.GetState(r.ctx, beacon.IdGenesis)
	if err != nil {
		return err
	}
	genPath := path.Join(e2e.TestParams.TestPath, "genesis.ssz")
	err = file.WriteFile(genPath, gb)
	if err != nil {
		return err
	}

	flags := append([]string{}, r.config.BeaconFlags...)
	flags = append(flags, fmt.Sprintf("--weak-subjectivity-checkpoint=%s", od.CheckpointString()))
	flags = append(flags, fmt.Sprintf("--checkpoint-state=%s", statePath))
	flags = append(flags, fmt.Sprintf("--checkpoint-block=%s", blockPath))
	flags = append(flags, fmt.Sprintf("--genesis-state=%s", genPath))

	//flags = append(flags, fmt.Sprintf("--checkpoint-sync-url=%s", bnAPI))
	//flags = append(flags, fmt.Sprintf("--genesis-beacon-api-url=%s", bnAPI))

	// zero-indexed, so next value would be len of list
	cpsyncer := components.NewBeaconNode(i, enr, flags, r.config)
	r.group.Go(func() error {
		return cpsyncer.Start(r.ctx)
	})
	if err := helpers.ComponentsStarted(r.ctx, []e2etypes.ComponentRunner{cpsyncer}); err != nil {
		return fmt.Errorf("checkpoint sync beacon node not ready: %w", err)
	}
	c, err := grpc.Dial(fmt.Sprintf("127.0.0.1:%d", e2e.TestParams.Ports.PrysmBeaconNodeRPCPort+i), grpc.WithInsecure())
	require.NoError(r.t, err, "Failed to dial")

	// this is so that the syncEvaluators checks can run on the checkpoint sync'd node
	conns = append(conns, c)
	err = r.waitForMatchingHead(r.ctx, c, conns[0])
	if err != nil {
		return err
	}

	syncEvaluators := []e2etypes.Evaluator{ev.FinishedSyncing, ev.AllNodesHaveSameHead}
	for _, evaluator := range syncEvaluators {
		r.t.Run(evaluator.Name, func(t *testing.T) {
			assert.NoError(t, evaluator.Evaluation(conns...), "Evaluation failed for sync node")
		})
	}
	return nil
}

// testBeaconChainSync creates another beacon node, and tests whether it can sync to head using previous nodes.
func (r *testRunner) testBeaconChainSync(ctx context.Context, g *errgroup.Group,
	conns []*grpc.ClientConn, tickingStartTime time.Time, bootnodeEnr, minerEnr string) error {
	index := e2e.TestParams.BeaconNodeCount + e2e.TestParams.LighthouseBeaconNodeCount
	t, config := r.t, r.config
	ethNode := eth1.NewNode(index, minerEnr)
	g.Go(func() error {
		return ethNode.Start(ctx)
	})
	if err := helpers.ComponentsStarted(ctx, []e2etypes.ComponentRunner{ethNode}); err != nil {
		return fmt.Errorf("sync beacon node not ready: %w", err)
	}
	syncBeaconNode := components.NewBeaconNode(index, bootnodeEnr, r.config.BeaconFlags, config)
	g.Go(func() error {
		return syncBeaconNode.Start(ctx)
	})
	if err := helpers.ComponentsStarted(ctx, []e2etypes.ComponentRunner{syncBeaconNode}); err != nil {
		return fmt.Errorf("sync beacon node not ready: %w", err)
	}
	syncConn, err := grpc.Dial(fmt.Sprintf("127.0.0.1:%d", e2e.TestParams.Ports.PrysmBeaconNodeRPCPort+index), grpc.WithInsecure())
	require.NoError(t, err, "Failed to dial")
	conns = append(conns, syncConn)

	// Sleep a second for every 4 blocks that need to be synced for the newly started node.
	secondsPerEpoch := uint64(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().SecondsPerSlot))
	extraSecondsToSync := (r.config.EpochsToRun)*secondsPerEpoch + uint64(params.BeaconConfig().SlotsPerEpoch.Div(4).Mul(r.config.EpochsToRun))
	waitForSync := tickingStartTime.Add(time.Duration(extraSecondsToSync) * time.Second)
	time.Sleep(time.Until(waitForSync))

	syncLogFile, err := os.Open(path.Join(e2e.TestParams.LogPath, fmt.Sprintf(e2e.BeaconNodeLogFileName, index)))
	require.NoError(r.t, err)
	defer helpers.LogErrorOutput(r.t, syncLogFile, "beacon chain node", index)
	r.t.Run("sync completed", func(t *testing.T) {
		assert.NoError(t, helpers.WaitForTextInFile(syncLogFile, "Synced up to"), "Failed to sync")
	})
	if r.t.Failed() {
		return errors.New("cannot sync beacon node")
	}

	// Sleep a slot to make sure the synced state is made.
	time.Sleep(time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Second)
	syncEvaluators := []e2etypes.Evaluator{ev.FinishedSyncing, ev.AllNodesHaveSameHead}
	for _, evaluator := range syncEvaluators {
		r.t.Run(evaluator.Name, func(t *testing.T) {
			assert.NoError(t, evaluator.Evaluation(conns...), "Evaluation failed for sync node")
		})
	}
	return nil
}

func (r *testRunner) testDoppelGangerProtection(ctx context.Context) error {
	// Exit if we are running from the previous release.
	if r.config.UsePrysmShValidator {
		return nil
	}
	g, ctx := errgroup.WithContext(ctx)
	// Follow same parameters as older validators.
	validatorNum := int(params.BeaconConfig().MinGenesisActiveValidatorCount)
	beaconNodeNum := e2e.TestParams.BeaconNodeCount
	if validatorNum%beaconNodeNum != 0 {
		return errors.New("validator count is not easily divisible by beacon node count")
	}
	validatorsPerNode := validatorNum / beaconNodeNum
	valIndex := beaconNodeNum + 1

	// Replicate starting up validator client 0 to test doppleganger protection.
	valNode := components.NewValidatorNode(r.config, validatorsPerNode, valIndex, validatorsPerNode*0)
	g.Go(func() error {
		return valNode.Start(ctx)
	})
	if err := helpers.ComponentsStarted(ctx, []e2etypes.ComponentRunner{valNode}); err != nil {
		return fmt.Errorf("validator not ready: %w", err)
	}
	logFile, err := os.Create(path.Join(e2e.TestParams.LogPath, fmt.Sprintf(e2e.ValidatorLogFileName, valIndex)))
	if err != nil {
		return fmt.Errorf("unable to open log file: %v", err)
	}
	r.t.Run("doppelganger found", func(t *testing.T) {
		assert.NoError(t, helpers.WaitForTextInFile(logFile, "Duplicate instances exists in the network for validator keys"), "Failed to carry out doppelganger check correctly")
	})
	if r.t.Failed() {
		return errors.New("doppelganger was unable to be found")
	}
	// Expect an abrupt exit for the validator client.
	if err := g.Wait(); err == nil || !strings.Contains(err.Error(), errGeneralCode) {
		return fmt.Errorf("wanted an error of %s but received %v", errGeneralCode, err)
	}
	return nil
}
