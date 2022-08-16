// Package endtoend performs full a end-to-end test for Prysm,
// including spinning up an ETH1 dev chain, sending deposits to the deposit
// contract, and making sure the beacon node and validators are running and
// performing properly for a few epochs.
package endtoend

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/v3/api/client/beacon"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v3/io/file"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/v3/proto/eth/service"
	v1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	eth "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/components"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/components/eth1"
	ev "github.com/prysmaticlabs/prysm/v3/testing/endtoend/evaluators"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/v3/testing/endtoend/params"
	e2etypes "github.com/prysmaticlabs/prysm/v3/testing/endtoend/types"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
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
	t          *testing.T
	config     *e2etypes.E2EConfig
	comHandler *componentHandler
}

// newTestRunner creates E2E test runner.
func newTestRunner(t *testing.T, config *e2etypes.E2EConfig) *testRunner {
	return &testRunner{
		t:      t,
		config: config,
	}
}

// run executes configured E2E test.
func (r *testRunner) run() {
	r.comHandler = NewComponentHandler(r.config, r.t)
	r.comHandler.setup()

	// Run E2E evaluators and tests.
	r.addEvent(r.defaultEndToEndRun)

	if err := r.comHandler.group.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		// At the end of the main evaluator goroutine all nodes are killed, no need to fail the test.
		if strings.Contains(err.Error(), "signal: killed") {
			return
		}
		r.t.Fatalf("E2E test ended in error: %v", err)
	}
}

func (r *testRunner) scenarioRunner() {
	r.comHandler = NewComponentHandler(r.config, r.t)
	r.comHandler.setup()

	// Run E2E evaluators and tests.
	r.addEvent(r.scenarioRun)

	if err := r.comHandler.group.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		// At the end of the main evaluator goroutine all nodes are killed, no need to fail the test.
		if strings.Contains(err.Error(), "signal: killed") {
			return
		}
		r.t.Fatalf("E2E test ended in error: %v", err)
	}
}

func (r *testRunner) waitExtra(ctx context.Context, e types.Epoch, conn *grpc.ClientConn, extra types.Epoch) error {
	spe := uint64(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().SecondsPerSlot))
	dl := time.Now().Add(time.Second * time.Duration(uint64(extra)*spe))

	beaconClient := eth.NewBeaconChainClient(conn)
	ctx, cancel := context.WithDeadline(ctx, dl)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return errors.Wrapf(ctx.Err(), "context deadline/cancel while waiting for epoch %d", e)
		default:
			chainHead, err := beaconClient.GetChainHead(ctx, &emptypb.Empty{})
			if err != nil {
				log.Warnf("while querying connection %s for chain head got error=%s", conn.Target(), err.Error())
			}
			if chainHead.HeadEpoch > e {
				// no need to wait, other nodes should be caught up
				return nil
			}
			if chainHead.HeadEpoch == e {
				// wait until halfway into the epoch to give other nodes time to catch up
				time.Sleep(time.Second * time.Duration(spe/2))
				return nil
			}
		}
	}
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
		if config.EvalInterceptor(currentEpoch, conns) {
			continue
		}
		r.executeProvidedEvaluators(currentEpoch, conns, config.Evaluators)

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
	// prysm web3signer doesn't support deposits
	r.config.UseWeb3RemoteSigner = false
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

func (r *testRunner) waitForMatchingHead(ctx context.Context, timeout time.Duration, check, ref *grpc.ClientConn) error {
	start := time.Now()
	dctx, cancel := context.WithDeadline(ctx, start.Add(timeout))
	defer cancel()
	checkClient := service.NewBeaconChainClient(check)
	refClient := service.NewBeaconChainClient(ref)
	for {
		select {
		case <-dctx.Done():
			// deadline ensures that the test eventually exits when beacon node fails to sync in a resonable timeframe
			elapsed := time.Since(start)
			return fmt.Errorf("deadline exceeded after %s waiting for known good block to appear in checkpoint-synced node", elapsed)
		default:
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

func (r *testRunner) testCheckpointSync(ctx context.Context, g *errgroup.Group, i int, conns []*grpc.ClientConn, bnAPI, enr, minerEnr string) error {
	matchTimeout := 3 * time.Minute
	ethNode := eth1.NewNode(i, minerEnr)
	g.Go(func() error {
		return ethNode.Start(ctx)
	})
	if err := helpers.ComponentsStarted(ctx, []e2etypes.ComponentRunner{ethNode}); err != nil {
		return fmt.Errorf("sync beacon node not ready: %w", err)
	}
	proxyNode := eth1.NewProxy(i)
	g.Go(func() error {
		return proxyNode.Start(ctx)
	})
	if err := helpers.ComponentsStarted(ctx, []e2etypes.ComponentRunner{proxyNode}); err != nil {
		return fmt.Errorf("sync beacon node not ready: %w", err)
	}

	client, err := beacon.NewClient(bnAPI)
	if err != nil {
		return err
	}
	gb, err := client.GetState(ctx, beacon.IdGenesis)
	if err != nil {
		return err
	}
	genPath := path.Join(e2e.TestParams.TestPath, "genesis.ssz")
	err = file.WriteFile(genPath, gb)
	if err != nil {
		return err
	}

	flags := append([]string{}, r.config.BeaconFlags...)
	flags = append(flags, fmt.Sprintf("--checkpoint-sync-url=%s", bnAPI))
	flags = append(flags, fmt.Sprintf("--genesis-beacon-api-url=%s", bnAPI))

	cfgcp := new(e2etypes.E2EConfig)
	*cfgcp = *r.config
	cfgcp.BeaconFlags = flags
	cpsyncer := components.NewBeaconNode(cfgcp, i, enr)
	g.Go(func() error {
		return cpsyncer.Start(ctx)
	})
	if err := helpers.ComponentsStarted(ctx, []e2etypes.ComponentRunner{cpsyncer}); err != nil {
		return fmt.Errorf("checkpoint sync beacon node not ready: %w", err)
	}
	c, err := grpc.Dial(fmt.Sprintf("127.0.0.1:%d", e2e.TestParams.Ports.PrysmBeaconNodeRPCPort+i), grpc.WithInsecure())
	require.NoError(r.t, err, "Failed to dial")

	// this is so that the syncEvaluators checks can run on the checkpoint sync'd node
	conns = append(conns, c)
	err = r.waitForMatchingHead(ctx, matchTimeout, c, conns[0])
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
	t, config := r.t, r.config
	index := e2e.TestParams.BeaconNodeCount + e2e.TestParams.LighthouseBeaconNodeCount
	ethNode := eth1.NewNode(index, minerEnr)
	g.Go(func() error {
		return ethNode.Start(ctx)
	})
	if err := helpers.ComponentsStarted(ctx, []e2etypes.ComponentRunner{ethNode}); err != nil {
		return fmt.Errorf("sync beacon node not ready: %w", err)
	}
	proxyNode := eth1.NewProxy(index)
	g.Go(func() error {
		return proxyNode.Start(ctx)
	})
	if err := helpers.ComponentsStarted(ctx, []e2etypes.ComponentRunner{proxyNode}); err != nil {
		return fmt.Errorf("sync beacon node not ready: %w", err)
	}
	syncBeaconNode := components.NewBeaconNode(config, index, bootnodeEnr)
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
	extraSecondsToSync := (config.EpochsToRun)*secondsPerEpoch + uint64(params.BeaconConfig().SlotsPerEpoch.Div(4).Mul(config.EpochsToRun))
	waitForSync := tickingStartTime.Add(time.Duration(extraSecondsToSync) * time.Second)
	time.Sleep(time.Until(waitForSync))

	syncLogFile, err := os.Open(path.Join(e2e.TestParams.LogPath, fmt.Sprintf(e2e.BeaconNodeLogFileName, index)))
	require.NoError(t, err)
	defer helpers.LogErrorOutput(t, syncLogFile, "beacon chain node", index)
	t.Run("sync completed", func(t *testing.T) {
		assert.NoError(t, helpers.WaitForTextInFile(syncLogFile, "Synced up to"), "Failed to sync")
	})
	if t.Failed() {
		return errors.New("cannot sync beacon node")
	}

	// Sleep a slot to make sure the synced state is made.
	time.Sleep(time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Second)
	syncEvaluators := []e2etypes.Evaluator{ev.FinishedSyncing, ev.AllNodesHaveSameHead}
	for _, evaluator := range syncEvaluators {
		t.Run(evaluator.Name, func(t *testing.T) {
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

func (r *testRunner) defaultEndToEndRun() error {
	t, config, ctx, g := r.t, r.config, r.comHandler.ctx, r.comHandler.group
	// When everything is done, cancel parent context (will stop all spawned nodes).
	defer func() {
		log.Info("All E2E evaluations are finished, cleaning up")
		r.comHandler.done()
	}()

	// Wait for all required nodes to start.
	ctxAllNodesReady, cancel := context.WithTimeout(ctx, allNodesStartTimeout)
	defer cancel()
	if err := helpers.ComponentsStarted(ctxAllNodesReady, r.comHandler.required()); err != nil {
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
	eth1Miner, ok := r.comHandler.eth1Miner.(*eth1.Miner)
	if !ok {
		return errors.New("incorrect component type")
	}
	beaconNodes, ok := r.comHandler.beaconNodes.(*components.BeaconNodeSet)
	if !ok {
		return errors.New("incorrect component type")
	}
	bootNode, ok := r.comHandler.bootnode.(*components.BootNode)
	if !ok {
		return errors.New("incorrect component type")
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

	index := e2e.TestParams.BeaconNodeCount + e2e.TestParams.LighthouseBeaconNodeCount
	if config.TestSync {
		if err := r.testBeaconChainSync(ctx, g, conns, tickingStartTime, bootNode.ENR(), eth1Miner.ENR()); err != nil {
			return errors.Wrap(err, "beacon chain sync test failed")
		}
		index += 1
		if err := r.testDoppelGangerProtection(ctx); err != nil {
			return errors.Wrap(err, "doppel ganger protection check failed")
		}
	}
	if config.TestCheckpointSync {
		httpEndpoints := helpers.BeaconAPIHostnames(e2e.TestParams.BeaconNodeCount)
		menr := eth1Miner.ENR()
		benr := bootNode.ENR()
		if err := r.testCheckpointSync(ctx, g, index, conns, httpEndpoints[0], benr, menr); err != nil {
			return errors.Wrap(err, "checkpoint sync test failed")
		}
		index += 1
	}

	if config.ExtraEpochs > 0 {
		if err := r.waitExtra(ctx, types.Epoch(config.EpochsToRun+config.ExtraEpochs), conns[0], types.Epoch(config.ExtraEpochs)); err != nil {
			return errors.Wrap(err, "error while waiting for ExtraEpochs")
		}
		syncEvaluators := []e2etypes.Evaluator{ev.FinishedSyncing, ev.AllNodesHaveSameHead}
		for _, evaluator := range syncEvaluators {
			t.Run(evaluator.Name, func(t *testing.T) {
				assert.NoError(t, evaluator.Evaluation(conns...), "Evaluation failed for sync node")
			})
		}
	}
	return nil
}

func (r *testRunner) scenarioRun() error {
	t, config, ctx := r.t, r.config, r.comHandler.ctx
	// When everything is done, cancel parent context (will stop all spawned nodes).
	defer func() {
		log.Info("All E2E evaluations are finished, cleaning up")
		r.comHandler.done()
	}()

	// Wait for all required nodes to start.
	ctxAllNodesReady, cancel := context.WithTimeout(ctx, allNodesStartTimeout)
	defer cancel()
	if err := helpers.ComponentsStarted(ctxAllNodesReady, r.comHandler.required()); err != nil {
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
	return r.runEvaluators(conns, tickingStartTime)
}
func (r *testRunner) addEvent(ev func() error) {
	r.comHandler.group.Go(ev)
}

func (r *testRunner) executeProvidedEvaluators(currentEpoch uint64, conns []*grpc.ClientConn, evals []e2etypes.Evaluator) {
	wg := new(sync.WaitGroup)
	for _, eval := range evals {
		// Fix reference to evaluator as it will be running
		// in a separate goroutine.
		evaluator := eval
		// Only run if the policy says so.
		if !evaluator.Policy(types.Epoch(currentEpoch)) {
			continue
		}
		wg.Add(1)
		go r.t.Run(fmt.Sprintf(evaluator.Name, currentEpoch), func(t *testing.T) {
			err := evaluator.Evaluation(conns...)
			assert.NoError(t, err, "Evaluation failed for epoch %d: %v", currentEpoch, err)
			wg.Done()
		})
	}
	wg.Wait()
}

// This interceptor will define the multi scenario run for our minimal tests.
// 1) In the first scenario we will be taking a single prysm node and its validator offline.
// Along with that we will also take a single lighthouse node and its validator offline.
// After 1 epoch we will then attempt to bring it online again.
//
//
// 2) Then we will start testing optimistic sync by engaging our engine proxy.
// After the proxy has been sending `SYNCING` responses to the beacon node, we
// will test this with our optimistic sync evaluator to ensure everything works
// as expected.
func (r *testRunner) multiScenarioMulticlient(epoch uint64, conns []*grpc.ClientConn) bool {
	type ForkchoiceUpdatedResponse struct {
		Status    *enginev1.PayloadStatus  `json:"payloadStatus"`
		PayloadId *enginev1.PayloadIDBytes `json:"payloadId"`
	}
	switch epoch {
	case 9:
		require.NoError(r.t, r.comHandler.beaconNodes.PauseAtIndex(0))
		require.NoError(r.t, r.comHandler.validatorNodes.PauseAtIndex(0))
		return true
	case 10:
		require.NoError(r.t, r.comHandler.beaconNodes.ResumeAtIndex(0))
		require.NoError(r.t, r.comHandler.validatorNodes.ResumeAtIndex(0))
		return true
	case 14:
		// Set it for prysm beacon node.
		component, err := r.comHandler.eth1Proxy.ComponentAtIndex(0)
		require.NoError(r.t, err)
		component.(e2etypes.EngineProxy).AddRequestInterceptor("engine_newPayloadV1", func() interface{} {
			return &enginev1.PayloadStatus{
				Status:          enginev1.PayloadStatus_SYNCING,
				LatestValidHash: make([]byte, 32),
			}
		}, func() bool {
			return true
		})
		// Set it for lighthouse beacon node.
		component, err = r.comHandler.eth1Proxy.ComponentAtIndex(2)
		require.NoError(r.t, err)
		component.(e2etypes.EngineProxy).AddRequestInterceptor("engine_newPayloadV1", func() interface{} {
			return &enginev1.PayloadStatus{
				Status:          enginev1.PayloadStatus_SYNCING,
				LatestValidHash: make([]byte, 32),
			}
		}, func() bool {
			return true
		})

		component.(e2etypes.EngineProxy).AddRequestInterceptor("engine_forkchoiceUpdatedV1", func() interface{} {
			return &ForkchoiceUpdatedResponse{
				Status: &enginev1.PayloadStatus{
					Status:          enginev1.PayloadStatus_SYNCING,
					LatestValidHash: nil,
				},
				PayloadId: nil,
			}
		}, func() bool {
			return true
		})
		return true
	case 15:
		r.executeProvidedEvaluators(epoch, []*grpc.ClientConn{conns[0]}, []e2etypes.Evaluator{
			ev.OptimisticSyncEnabled,
		})
		// Disable Interceptor
		component, err := r.comHandler.eth1Proxy.ComponentAtIndex(0)
		require.NoError(r.t, err)
		engineProxy, ok := component.(e2etypes.EngineProxy)
		require.Equal(r.t, true, ok)
		engineProxy.RemoveRequestInterceptor("engine_newPayloadV1")
		engineProxy.ReleaseBackedUpRequests("engine_newPayloadV1")

		// Remove for lighthouse too
		component, err = r.comHandler.eth1Proxy.ComponentAtIndex(2)
		require.NoError(r.t, err)
		engineProxy, ok = component.(e2etypes.EngineProxy)
		require.Equal(r.t, true, ok)
		engineProxy.RemoveRequestInterceptor("engine_newPayloadV1")
		engineProxy.RemoveRequestInterceptor("engine_forkchoiceUpdatedV1")
		engineProxy.ReleaseBackedUpRequests("engine_newPayloadV1")

		return true
	case 11, 12, 16, 17:
		// Allow 2 epochs for the network to finalize again.
		return true
	}
	return false
}

func (r *testRunner) eeOffline(epoch uint64, _ []*grpc.ClientConn) bool {
	switch epoch {
	case 9:
		require.NoError(r.t, r.comHandler.eth1Miner.Pause())
		return true
	case 10:
		require.NoError(r.t, r.comHandler.eth1Miner.Resume())
		return true
	case 11, 12:
		// Allow 2 epochs for the network to finalize again.
		return true
	}
	return false
}

// This interceptor will define the multi scenario run for our minimal tests.
// 1) In the first scenario we will be taking a single node and its validator offline.
// After 1 epoch we will then attempt to bring it online again.
//
// 2) In the second scenario we will be taking all validators offline. After 2
// epochs we will wait for the network to recover.
//
// 3) Then we will start testing optimistic sync by engaging our engine proxy.
// After the proxy has been sending `SYNCING` responses to the beacon node, we
// will test this with our optimistic sync evaluator to ensure everything works
// as expected.
func (r *testRunner) multiScenario(epoch uint64, conns []*grpc.ClientConn) bool {
	switch epoch {
	case 9:
		require.NoError(r.t, r.comHandler.beaconNodes.PauseAtIndex(0))
		require.NoError(r.t, r.comHandler.validatorNodes.PauseAtIndex(0))
		return true
	case 10:
		require.NoError(r.t, r.comHandler.beaconNodes.ResumeAtIndex(0))
		require.NoError(r.t, r.comHandler.validatorNodes.ResumeAtIndex(0))
		return true
	case 14:
		require.NoError(r.t, r.comHandler.validatorNodes.PauseAtIndex(0))
		require.NoError(r.t, r.comHandler.validatorNodes.PauseAtIndex(1))
		return true
	case 15:
		require.NoError(r.t, r.comHandler.validatorNodes.ResumeAtIndex(0))
		require.NoError(r.t, r.comHandler.validatorNodes.ResumeAtIndex(1))
		return true
	case 19:
		component, err := r.comHandler.eth1Proxy.ComponentAtIndex(0)
		require.NoError(r.t, err)
		component.(e2etypes.EngineProxy).AddRequestInterceptor("engine_newPayloadV1", func() interface{} {
			return &enginev1.PayloadStatus{
				Status:          enginev1.PayloadStatus_SYNCING,
				LatestValidHash: make([]byte, 32),
			}
		}, func() bool {
			return true
		})
		return true
	case 20:
		r.executeProvidedEvaluators(epoch, []*grpc.ClientConn{conns[0]}, []e2etypes.Evaluator{
			ev.OptimisticSyncEnabled,
		})
		// Disable Interceptor
		component, err := r.comHandler.eth1Proxy.ComponentAtIndex(0)
		require.NoError(r.t, err)
		engineProxy, ok := component.(e2etypes.EngineProxy)
		require.Equal(r.t, true, ok)
		engineProxy.RemoveRequestInterceptor("engine_newPayloadV1")
		engineProxy.ReleaseBackedUpRequests("engine_newPayloadV1")

		return true
	case 11, 12, 16, 17, 21, 22:
		// Allow 2 epochs for the network to finalize again.
		return true
	}
	return false
}

// All Epochs are valid.
func defaultInterceptor(_ uint64, _ []*grpc.ClientConn) bool {
	return false
}
