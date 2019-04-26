package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	gethRPC "github.com/ethereum/go-ethereum/rpc"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/attestation"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"github.com/x-cray/logrus-prefixed-formatter"
)

var log = logrus.WithField("prefix", "state-replay")

type mockP2P struct{}

func (m *mockP2P) Broadcast(ctx context.Context, msg proto.Message) {}

func main() {
	var dbPath = flag.String("db-dir", "", "path to bolt.db dir")
	flag.Parse()

	customFormatter := new(prefixed.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	logrus.SetFormatter(customFormatter)
	//logrus.SetLevel(logrus.DebugLevel)

	readOnlyDB, err := db.NewReadOnlyDB(*dbPath)
	if err != nil {
		log.Fatal(err)
	}
	beaconDB, err := db.NewDB("/tmp/state-replay-dir")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll("/tmp/state-replay-dir")
	ctx := context.Background()
	params.UseDemoBeaconConfig()
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{})

	// Setup a chain service and powchain service.
	rpcClient, err := gethRPC.Dial("wss://goerli.prylabs.net/websocket")
	if err != nil {
		log.Fatalf("Access to PoW chain is required for validator. Unable to connect to Geth node: %v", err)
	}
	httpRPCClient, err := gethRPC.Dial("https://goerli.prylabs.net")
	if err != nil {
		log.Fatalf("Access to PoW chain is required for validator. Unable to connect to Geth node: %v", err)
	}
	powClient := ethclient.NewClient(rpcClient)
	httpClient := ethclient.NewClient(httpRPCClient)
	cfg := &powchain.Web3ServiceConfig{
		Endpoint:        "wss://goerli.prylabs.net/websocket",
		DepositContract: common.HexToAddress("0x30b3366A1c57F124b9B8fD17d95f97d5363Da6a6"),
		Client:          powClient,
		Reader:          powClient,
		Logger:          powClient,
		HTTPLogger: httpClient,
		BlockFetcher:    powClient,
		ContractBackend: powClient,
		BeaconDB:        beaconDB,
	}
	web3Service, err := powchain.NewWeb3Service(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	attService := attestation.NewAttestationService(ctx, &attestation.Config{
		BeaconDB: beaconDB,
	})
	opsService := operations.NewOpsPoolService(ctx, &operations.Config{
		BeaconDB: beaconDB,
	})
	chainService, err := blockchain.NewChainService(ctx, &blockchain.Config{
		BeaconDB:    beaconDB,
		P2p: &mockP2P{},
		OpsPoolService: opsService,
		AttsService: attService,
		Web3Service: web3Service,
	})
	if err != nil {
		log.Fatal(err)
	}

	attesterServer := rpc.NewAttesterServer(beaconDB)

	stateInit := make(chan time.Time)
	stateInitFeed := chainService.StateInitializedFeed()
	stateInitFeed.Subscribe(stateInit)

	chainService.Start()
	defer chainService.Stop()
	opsService.Start()
	defer opsService.Stop()
	attService.Start()
	defer attService.Stop()
	web3Service.Start()
	defer web3Service.Stop()

	log.Info("Waiting for chainstart...")
	<-stateInit

	// Begin the replay of the system.
	// Get the highest information.
	highestState, err := readOnlyDB.HeadState(ctx)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("Highest state: %d, current state: %d", highestState.Slot-params.BeaconConfig().GenesisSlot, 0)

	lastFinalizedState, err := readOnlyDB.FinalizedState()
	if err != nil {
		log.Fatal(err)
	}
	if err := beaconDB.SaveBlock(lastFinalizedState.LatestBlock); err != nil {
		log.Fatal(err)
	}
	if err := beaconDB.UpdateChainHead(ctx, lastFinalizedState.LatestBlock, lastFinalizedState); err != nil {
		log.Fatal(err)
	}

	currentState := lastFinalizedState
	for currentSlot := currentState.Slot + 1; currentSlot <= highestState.Slot; currentSlot++ {
		log.Infof("Slot %d", currentSlot-params.BeaconConfig().GenesisSlot)
		newBlock, err := readOnlyDB.BlockBySlot(ctx, currentSlot)
		if err != nil {
			log.Fatal(err)
		}
		if newBlock == nil {
			log.Warnf("No block at slot %d", currentSlot-params.BeaconConfig().GenesisSlot)
			continue
		}
		if newBlock.Slot == params.BeaconConfig().GenesisSlot+47 {
			continue
		}

		newState, err := chainService.ApplyBlockStateTransition(ctx, newBlock, currentState)
		if err != nil {
			att := newBlock.Body.Attestations[0]
			log.Infof("Slot: %v", att.Data.Slot-params.BeaconConfig().GenesisSlot)
			log.Infof("Justified epoch: %v", att.Data.JustifiedEpoch-params.BeaconConfig().GenesisEpoch)
			log.Infof("Block root: %#x", att.Data.BeaconBlockRootHash32)
			log.Infof("Epoch boundary root: %#x", att.Data.EpochBoundaryRootHash32)
			log.Infof("Justified block root: %#x", att.Data.JustifiedBlockRootHash32)
			log.Fatalf("Could not apply state transition: %v", err)
		}
		if err := chainService.CleanupBlockOperations(ctx, newBlock); err != nil {
            log.Fatal(err)
		}
		if err := beaconDB.SaveBlock(newBlock); err != nil {
			log.Fatal(err)
		}
		if err := beaconDB.UpdateChainHead(ctx, newBlock, newState); err != nil {
			log.Fatalf("Could not update chain head: %v", err)
		}
		root, err := hashutil.HashBeaconBlock(newBlock)
		if err != nil {
			log.Fatal(err)
		}
		log.Infof("Saved block with root: %#x", root)
		attInfo, err := attesterServer.AttestationDataAtSlot(ctx, &pb.AttestationDataRequest{
			Slot: newBlock.Slot,
		})
		if err != nil {
			log.Fatal(err)
		}
		log.Info("")
		log.Info("----Attestation Data At Slot----")
		log.Infof("Head state slot: %v", attInfo.HeadSlot-params.BeaconConfig().GenesisSlot)
		log.Infof("Justified epoch: %v", attInfo.JustifiedEpoch-params.BeaconConfig().GenesisEpoch)
		log.Infof("Justified root hash: %#x", attInfo.JustifiedBlockRootHash32)
		log.Infof("Epoch boundary root hash: %#x", attInfo.EpochBoundaryRootHash32)
		log.Info("---------------------------------")
		log.Info("")
		currentState = newState
	}
	log.Info(currentState.Slot)
}
