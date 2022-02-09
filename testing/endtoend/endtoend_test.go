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

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/proto/eth/service"
	v1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	eth2 "github.com/prysmaticlabs/prysm/proto/eth/v2"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/sniff"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/endtoend/components"
	ev "github.com/prysmaticlabs/prysm/testing/endtoend/evaluators"
	"github.com/prysmaticlabs/prysm/testing/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/testing/endtoend/params"
	e2etypes "github.com/prysmaticlabs/prysm/testing/endtoend/types"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/time/slots"
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
	t        *testing.T
	config   *e2etypes.E2EConfig
	ctx      context.Context
	doneChan context.CancelFunc
	group    *errgroup.Group
}

// newTestRunner creates E2E test runner.
func newTestRunner(t *testing.T, config *e2etypes.E2EConfig) *testRunner {
	ctx, done := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)
	return &testRunner{
		ctx:      ctx,
		doneChan: done,
		group:    g,
		t:        t,
		config:   config,
	}
}

// run executes configured E2E test.
func (r *testRunner) run() {
	t, config := r.t, r.config
	t.Logf("Shard index: %d\n", e2e.TestParams.TestShardIndex)
	t.Logf("Starting time: %s\n", time.Now().String())
	t.Logf("Log Path: %s\n", e2e.TestParams.LogPath)

	// we need debug turned on and max ssz payload bumped up when running checkpoint sync tests
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

	if multiClientActive {
		keyGen = components.NewKeystoreGenerator()

		// Generate lighthouse keystores.
		g.Go(func() error {
			return keyGen.Start(ctx)
		})
	}

	// ETH1 node.
	eth1Node := components.NewEth1Node()
	g.Go(func() error {
		if err := eth1Node.Start(ctx); err != nil {
			return errors.Wrap(err, "failed to start eth1node")
		}
		return nil
	})
	g.Go(func() error {
		if err := helpers.ComponentsStarted(ctx, []e2etypes.ComponentRunner{eth1Node}); err != nil {
			return errors.Wrap(err, "sending and mining deposits require ETH1 node to run")
		}
		if err := components.SendAndMineDeposits(eth1Node.KeystorePath(), minGenesisActiveCount, 0, true /* partial */); err != nil {
			return errors.Wrap(err, "failed to send and mine deposits")
		}
		return nil
	})

	// Boot node.
	bootNode := components.NewBootNode()
	g.Go(func() error {
		if err := bootNode.Start(ctx); err != nil {
			return errors.Wrap(err, "failed to start bootnode")
		}
		return nil
	})
	// Beacon nodes.
	beaconNodes := components.NewBeaconNodes(config.BeaconFlags, config)
	g.Go(func() error {
		if err := helpers.ComponentsStarted(ctx, []e2etypes.ComponentRunner{eth1Node, bootNode}); err != nil {
			return errors.Wrap(err, "beacon nodes require ETH1 and boot node to run")
		}
		beaconNodes.SetENR(bootNode.ENR())
		if err := beaconNodes.Start(ctx); err != nil {
			return errors.Wrap(err, "failed to start beacon nodes")
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

	if multiClientActive {
		lighthouseNodes = components.NewLighthouseBeaconNodes(config)
		g.Go(func() error {
			if err := helpers.ComponentsStarted(ctx, []e2etypes.ComponentRunner{eth1Node, bootNode, beaconNodes}); err != nil {
				return errors.Wrap(err, "lighthouse beacon nodes require ETH1 and boot node to run")
			}
			lighthouseNodes.SetENR(bootNode.ENR())
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
			tracingSink, eth1Node, bootNode, beaconNodes, validatorNodes,
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

		if config.TestDeposits {
			log.Info("Running deposit tests")
			r.testDeposits(ctx, g, eth1Node, []e2etypes.ComponentRunner{beaconNodes})
		}

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

		// If requested, run sync test.
		if !config.TestSync {
			return nil
		}
		index := e2e.TestParams.BeaconNodeCount
		if err := r.testBeaconChainSync(index, conns, tickingStartTime, bootNode.ENR()); err != nil {
			return errors.Wrap(err, "beacon chain sync test failed")
		}
		if err := r.testCheckpointSync(index+1, conns, tickingStartTime, bootNode.ENR()); err != nil {
			return errors.Wrap(err, "checkpoint sync test failed")
		}
		if err := r.testDoppelGangerProtection(ctx); err != nil {
			return errors.Wrap(err, "doppel ganger protection check failed")
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

// testDeposits runs tests when config.TestDeposits is enabled.
func (r *testRunner) testDeposits(ctx context.Context, g *errgroup.Group,
	eth1Node *components.Eth1Node, requiredNodes []e2etypes.ComponentRunner) {
	minGenesisActiveCount := int(params.BeaconConfig().MinGenesisActiveValidatorCount)

	depositCheckValidator := components.NewValidatorNode(r.config, int(e2e.DepositCount), e2e.TestParams.BeaconNodeCount, minGenesisActiveCount)
	g.Go(func() error {
		if err := helpers.ComponentsStarted(ctx, requiredNodes); err != nil {
			return fmt.Errorf("deposit check validator node requires beacon nodes to run: %w", err)
		}
		go func() {
			err := components.SendAndMineDeposits(eth1Node.KeystorePath(), int(e2e.DepositCount), minGenesisActiveCount, false /* partial */)
			if err != nil {
				r.t.Fatal(err)
			}
		}()
		return depositCheckValidator.Start(ctx)
	})
}

type saveable interface {
	MarshalSSZ() ([]byte, error)
}

func saveSSZBytes(filePath string, value saveable) (err error) {
	b, err := value.MarshalSSZ()
	if err != nil {
		err = errors.Wrap(err, "saveSSZBytes/MarshalSSZ")
		return err
	}
	fh, err := os.Create(filePath)
	if err != nil {
		err = errors.Wrap(err, "saveSSZBytes/os.Create")
		return err
	}
	defer func() {
		err = errors.Wrap(err, "saveSSZBytes/defered close")
		err = fh.Close()
	}()
	_, err = fh.Write(b)
	if err != nil {
		err = errors.Wrap(err, "saveSSZBytes/fh.Write")
	}
	return err
}

func saveBlock(ctx context.Context, conn *grpc.ClientConn, cf *sniff.ConfigFork, root [32]byte, basePath string) (string, error) {
	v1Client := service.NewBeaconChainClient(conn)
	//blockId := fmt.Sprintf("%#x", root)
	bResp, err := v1Client.GetBlockSSZV2(ctx, &eth2.BlockRequestV2{BlockId: root[:]})
	if err != nil {
		err = errors.Wrap(err, "saveBlock/GetBeaconBlock")
		return "", err
	}
	sb, err := sniff.BlockForConfigFork(bResp.GetData(), cf)
	if err != nil {
		err = errors.Wrap(err, "saveBlock/GetBeaconBlock")
		return "", err
	}

	p := path.Join(basePath, fmt.Sprintf("block_%d_%x.ssz", sb.Block().Slot(), root))
	err = saveSSZBytes(p, sb)
	if err != nil {
		err = errors.Wrap(err, "saveBlock/saveSSZBytes")
	}
	return p, err
}

func getConfigFork(ctx context.Context, conn *grpc.ClientConn, slot types.Slot) (*sniff.ConfigFork, error) {
	ofs, err := getOrderedForkSchedule(ctx, conn)
	if err != nil {
		err = errors.Wrap(err, "getConfigFork/getOrderedForkSchedule")
		return nil, err
	}
	epoch := slots.ToEpoch(slot)
	version, err := ofs.VersionForEpoch(epoch)
	if err != nil {
		err = errors.Wrap(err, "getConfigFork/VersionForEpoch")
		return nil, err
	}
	cf, err := sniff.FindConfigFork(version)
	if err != nil {
		err = errors.Wrap(err, "getConfigFork/FindConfigFork")
		return nil, err
	}
	return cf, nil
}

func saveState(ctx context.Context, conn *grpc.ClientConn, cf *sniff.ConfigFork, slot types.Slot, basePath string) (string, [32]byte, error) {
	debugClient := service.NewBeaconDebugClient(conn)
	stateId := []byte(fmt.Sprintf("%d", slot))
	sResp, err := debugClient.GetBeaconStateSSZV2(ctx, &eth2.StateRequestV2{StateId: stateId})
	if err != nil {
		err = errors.Wrap(err, "saveState/GetBeaconState")
		return "", [32]byte{}, err
	}
	state, err := sniff.BeaconStateForConfigFork(sResp.Data, cf)
	if err != nil {
		return "", [32]byte{}, errors.Wrap(err, "saveState/BeaconStateForConfigFork")
	}
	stateRoot, err := state.HashTreeRoot(ctx)
	if err != nil {
		return "", [32]byte{}, errors.Wrap(err, "saveState/BeaconStateForConfigFork")
	}

	p := path.Join(basePath, fmt.Sprintf("state_%d_%x.ssz", state.Slot(), stateRoot))
	err = saveSSZBytes(p, state)
	if err != nil {
		err = errors.Wrap(err, "saveState/saveSSZBytes")
	}
	return p, stateRoot, err
}

type checkpoint struct {
	statePath string
	stateRoot [32]byte
	blockPath string
	blockRoot [32]byte
	headRoot  [32]byte
	epoch     types.Epoch
}

func (c *checkpoint) flags() []string {
	return []string{
		fmt.Sprintf("--checkpoint-state=%s", c.statePath),
		fmt.Sprintf("--checkpoint-block=%s", c.blockPath),
		fmt.Sprintf("--weak-subjectivity-checkpoint=%x:%d", c.blockRoot, c.epoch),
	}
}

func getOrderedForkSchedule(ctx context.Context, conn *grpc.ClientConn) (params.OrderedForkSchedule, error) {
	v1Client := service.NewBeaconChainClient(conn)
	fsResp, err := v1Client.GetForkSchedule(ctx, &emptypb.Empty{})
	if err != nil {
		err = errors.Wrap(err, "getOrderedForkSchedule:GetForkSchedule")
		return nil, err
	}
	ofs := make(params.OrderedForkSchedule, 0)
	for _, f := range fsResp.Data {
		ofs = append(ofs, params.ForkScheduleEntry{
			Version: bytesutil.ToBytes4(f.CurrentVersion),
			Epoch:   f.Epoch,
		})
	}
	return ofs, nil
}

func getHeadBlockRoot(ctx context.Context, conn *grpc.ClientConn) ([32]byte, error) {
	v1Client := service.NewBeaconChainClient(conn)
	bResp, err := v1Client.GetBlockRoot(ctx, &v1.BlockRequest{BlockId: []byte("head")})
	if err != nil {
		err = errors.Wrap(err, "getHeadBlockRoot:GetBlockRoot")
		return [32]byte{}, err
	}
	return bytesutil.ToBytes32(bResp.Data.Root), nil
}

func DownloadCheckpoint(ctx context.Context, conn *grpc.ClientConn) (*checkpoint, error) {
	v1Client := service.NewBeaconChainClient(conn)
	resp, err := v1Client.GetWeakSubjectivity(ctx, &emptypb.Empty{})
	if err != nil {
		err = errors.Wrap(err, "DownloadCheckpoint:GetWeakSubjectivityCheckpointEpoch")
		return nil, err
	}
	ws := resp.Data
	cp := &checkpoint{
		epoch:     ws.WsCheckpoint.Epoch,
		stateRoot: bytesutil.ToBytes32(ws.StateRoot),
		blockRoot: bytesutil.ToBytes32(ws.WsCheckpoint.Root),
	}

	headRoot, err := getHeadBlockRoot(ctx, conn)
	if err != nil {
		err = errors.Wrap(err, "DownloadCheckpoint:getHeadBlockRoot")
		return nil, err
	}
	cp.headRoot = headRoot

	// save the block at epoch start slot
	wsSlot, err := slots.EpochStart(cp.epoch)
	if err != nil {
		err = errors.Wrap(err, "DownloadCheckpoint:EpochStart")
		return nil, err
	}

	// fetch the state for the slot immediately following (and therefore integrating) the block
	cf, err := getConfigFork(ctx, conn, wsSlot)
	if err != nil {
		err = errors.Wrap(err, "DownloadCheckpoint:getConfigFork")
		return nil, err
	}

	cp.blockPath, err = saveBlock(ctx, conn, cf, cp.blockRoot, e2e.TestParams.TestPath)
	if err != nil {
		err = errors.Wrap(err, "DownloadCheckpoint:saveBlock")
		return nil, err
	}

	var sr [32]byte
	cp.statePath, sr, err = saveState(ctx, conn, cf, wsSlot, e2e.TestParams.TestPath)
	if err != nil {
		err = errors.Wrap(err, "DownloadCheckpoint:saveState")
		return nil, err
	}
	if sr != cp.stateRoot {
		err = fmt.Errorf("state htr (%#x) at slot %d != weak_subjectivity response (%#x)", cp.stateRoot, wsSlot, sr)
		return nil, err
	}

	return cp, nil
}

func (r *testRunner) waitForSentinelBlock(ctx context.Context, conn *grpc.ClientConn, root [32]byte) error {
	// sleep hack copied from testBeaconChainSync
	// Sleep a second for every 4 blocks that need to be synced for the newly started node.
	secondsPerEpoch := uint64(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().SecondsPerSlot))
	extraSecondsToSync := (r.config.EpochsToRun)*secondsPerEpoch + uint64(params.BeaconConfig().SlotsPerEpoch.Div(4).Mul(r.config.EpochsToRun))
	ctx, cancel := context.WithDeadline(r.ctx, time.Now().Add(time.Second*time.Duration(extraSecondsToSync)))
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			// deadline ensures that the test eventually exits when beacon node fails to sync in a resonable timeframe
			return fmt.Errorf("deadline exceeded waiting for known good block to appear in checkpoint-synced node")
		case <-time.After(time.Second * 1):
			v1Client := service.NewBeaconChainClient(conn)
			bResp, err := v1Client.GetBlockRoot(ctx, &v1.BlockRequest{BlockId: []byte("head")})
			if err != nil {
				errStatus, ok := status.FromError(err)
				// in the happy path we expect NotFound results until the node has synced
				if ok && errStatus.Code() == codes.NotFound {
					continue
				}

				return fmt.Errorf("error requesting block w/ root '%x' = %s", root, err)
			}
			// we have a match, sentinel block found
			if bytesutil.ToBytes32(bResp.Data.Root) == root {
				return nil
			}
		}
	}
}

func (r *testRunner) testCheckpointSync(i int, conns []*grpc.ClientConn, tickingStartTime time.Time, enr string) error {
	conn := conns[0]
	cp, err := DownloadCheckpoint(r.ctx, conn)
	if err != nil {
		return err
	}
	flags := append(r.config.BeaconFlags, cp.flags()...)
	// zero-indexed, so next value would be len of list
	cpsyncer := components.NewBeaconNode(i, enr, flags, r.config)
	r.group.Go(func() error {
		return cpsyncer.Start(r.ctx)
	})
	if err := helpers.ComponentsStarted(r.ctx, []e2etypes.ComponentRunner{cpsyncer}); err != nil {
		return fmt.Errorf("checkpoint sync beacon node not ready: %w", err)
	}
	c, err := grpc.Dial(fmt.Sprintf("127.0.0.1:%d", e2e.TestParams.BeaconNodeRPCPort+i), grpc.WithInsecure())
	require.NoError(r.t, err, "Failed to dial")

	// this is so that the syncEvaluators checks can run on the checkpoint sync'd node
	conns = append(conns, c)
	err = r.waitForSentinelBlock(r.ctx, conn, cp.headRoot)
	if err != nil {
		return err
	}

	syncLogFile, err := os.Open(path.Join(e2e.TestParams.LogPath, fmt.Sprintf(e2e.BeaconNodeLogFileName, i)))
	require.NoError(r.t, err)
	defer helpers.LogErrorOutput(r.t, syncLogFile, "beacon chain node", i)
	r.t.Run("sync completed", func(t *testing.T) {
		assert.NoError(t, helpers.WaitForTextInFile(syncLogFile, "Synced up to"), "Failed to sync")
	})
	if r.t.Failed() {
		return errors.New("cannot sync beacon node")
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
func (r *testRunner) testBeaconChainSync(index int, conns []*grpc.ClientConn, tickingStartTime time.Time, enr string) error {
	syncBeaconNode := components.NewBeaconNode(index, enr, r.config.BeaconFlags, r.config)
	r.group.Go(func() error {
		return syncBeaconNode.Start(r.ctx)
	})
	if err := helpers.ComponentsStarted(r.ctx, []e2etypes.ComponentRunner{syncBeaconNode}); err != nil {
		return fmt.Errorf("sync beacon node not ready: %w", err)
	}
	syncConn, err := grpc.Dial(fmt.Sprintf("127.0.0.1:%d", e2e.TestParams.BeaconNodeRPCPort+index), grpc.WithInsecure())
	require.NoError(r.t, err, "Failed to dial")
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
