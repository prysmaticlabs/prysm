package main

import (
	"context"
	"flag"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	gethRPC "github.com/ethereum/go-ethereum/rpc"
	"github.com/prysmaticlabs/prysm/beacon-chain/attestation"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"time"
)

var log = logrus.WithField("prefix", "state-replay")

func main() {
	var dbPath = flag.String("db-dir", "", "path to bolt.db dir")
	flag.Parse()
	readOnlyDB, err := db.NewReadOnlyDB(*dbPath)
	if err != nil {
		log.Fatal(err)
	}
	beaconDB, err := db.NewDB("/tmp/state-replay-dir-0")
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	params.UseDemoBeaconConfig()
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{})

	// Setup a chain service and powchain service.
	rpcClient, err := gethRPC.Dial("wss://goerli.prylabs.net/websocket")
	if err != nil {
		log.Fatalf("Access to PoW chain is required for validator. Unable to connect to Geth node: %v", err)
	}
	powClient := ethclient.NewClient(rpcClient)
	cfg := &powchain.Web3ServiceConfig{
		Endpoint:        "wss://goerli.prylabs.net/websocket",
		DepositContract: common.HexToAddress("0x10312bc0Cd24Ad27971f741F265818aA523db27b"),
		Client:          powClient,
		Reader:          powClient,
		Logger:          powClient,
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
	chainService, err := blockchain.NewChainService(ctx, &blockchain.Config{
		BeaconDB:    beaconDB,
		AttsService: attService,
		Web3Service: web3Service,
	})
	if err != nil {
		log.Fatal(err)
	}

	stateInit := make(chan time.Time)
	stateInitFeed := chainService.StateInitializedFeed()
	stateInitFeed.Subscribe(stateInit)

	chainService.Start()
	defer chainService.Stop()
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

	genesisState, err := beaconDB.HeadState(ctx)
	if err != nil {
		log.Fatal(err)
	}

	currentState := genesisState
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

		newState, err := chainService.ApplyBlockStateTransition(ctx, newBlock, currentState)
		if err != nil {
			log.Fatalf("Could not apply state transition: %v", err)
		}
		if err := beaconDB.SaveBlock(newBlock); err != nil {
			log.Fatal(err)
		}
		if err := beaconDB.UpdateChainHead(ctx, newBlock, newState); err != nil {
			log.Fatalf("Could not update chain head: %v", err)
		}

		currentState = newState
	}
	log.Info(currentState.Slot)
}
