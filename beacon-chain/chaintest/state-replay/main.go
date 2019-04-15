package main

import (
	"context"
	"flag"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "state-replay")

func main() {
	var dbPath = flag.String("db-dir", "", "path to bolt.db dir")
	flag.Parse()
	readOnlyDB, err := db.NewReadOnlyDB(*dbPath)
	if err != nil {
		log.Fatal(err)
	}
	beaconDB, err := db.NewDB("/tmp/state-replay-dir")
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	params.UseDemoBeaconConfig()
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{})
	chainService, err := blockchain.NewChainService(ctx, &blockchain.Config{
		BeaconDB:    beaconDB,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Begin the replay of the system.
	// Get the highest information.
	highestState, err := readOnlyDB.HeadState(ctx)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("Checking historical genesis state...")
	genesisState, err := readOnlyDB.HistoricalStateFromSlot(ctx, params.BeaconConfig().GenesisSlot)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("Highest state: %d, current state: %d", highestState.Slot-params.BeaconConfig().GenesisSlot, 0)

	genesisBlock, err := readOnlyDB.BlockBySlot(ctx, params.BeaconConfig().GenesisSlot)
	if err != nil {
		log.Fatal(err)
	}
	log.Info(genesisBlock)

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
		if err := beaconDB.UpdateChainHead(ctx, newBlock, newState); err != nil {
			log.Fatalf("Could not update chain head: %v", err)
		}

		currentState = newState
	}
	log.Info(currentState.Slot)
	//f, err := os.Create("/tmp/state1")
	//if err != nil {
	//	log.Fatal(err)
	//}
	//defer f.Close()
	//buf := new(bytes.Buffer)
	//if err := proto.MarshalText(buf, currentState); err != nil {
	//	log.Fatal(err)
	//}
	//s := buf.String()
	//if _, err := f.WriteString(s); err != nil {
	//	log.Fatal(err)
	//}
}
