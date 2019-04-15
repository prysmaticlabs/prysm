package main

import (
"bytes"
"context"
"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
"os"

"github.com/gogo/protobuf/proto"
"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
"github.com/prysmaticlabs/prysm/beacon-chain/db"
"github.com/prysmaticlabs/prysm/beacon-chain/rpc"
pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
"github.com/prysmaticlabs/prysm/shared/featureconfig"
"github.com/prysmaticlabs/prysm/shared/params"
"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "state-replay")

func main() {
	dbRO, err := db.NewDB("node1dir")
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	params.UseDemoBeaconConfig()
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{})
	chainService, err := blockchain.NewChainService(ctx, &blockchain.Config{
		BeaconDB:    dbRO,
	})
	if err != nil {
		log.Fatal(err)
	}
	validatorServer := rpc.NewValidatorServer(ctx, &rpc.ValidatorConfig{
		ChainService: chainService,
		BeaconDB: dbRO,
	})

	// Begin the replay of the system.
	// Get the highest information.
	highestState, err := dbRO.HeadState(ctx)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("Checking historical genesis state")
	genesisState, err := dbRO.HistoricalStateFromSlot(ctx, params.BeaconConfig().GenesisSlot)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("Highest state: %d, current state: %d", highestState.Slot-params.BeaconConfig().GenesisSlot, 0)

	genesisBlockRO, err := dbRO.BlockBySlot(ctx, params.BeaconConfig().GenesisSlot)
	if err != nil {
		log.Fatal(err)
	}
	log.Info(genesisBlockRO)

	currentState := genesisState
	for currentSlot := currentState.Slot + 1; currentSlot <= highestState.Slot; currentSlot++ {
		log.Infof("Slot %d", currentSlot-params.BeaconConfig().GenesisSlot)
		newBlock, err := dbRO.BlockBySlot(ctx, currentSlot)
		if err != nil {
			log.Fatal(err)
		}
		if newBlock == nil {
			log.Warnf("no block at slot %d", currentSlot)
			continue
		}

		newState, err := chainService.ApplyBlockStateTransition(ctx, newBlock, currentState)
		if err != nil {
			log.Fatal(err)
		}
		if err := dbRO.UpdateChainHead(ctx, newBlock, newState); err != nil {
			log.Fatal(err)
		}

		currentState = newState
	}
	f, err := os.Create("/tmp/state1")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	buf := new(bytes.Buffer)
	if err := proto.MarshalText(buf, currentState); err != nil {
		log.Fatal(err)
	}
	s := buf.String()
	if _, err := f.WriteString(s); err != nil {
		log.Fatal(err)
	}

	// Finally, we gather the public keys.
	var keys [][]byte
	activeVals := helpers.ActiveValidatorIndices(currentState.ValidatorRegistry, helpers.SlotToEpoch(currentState.Slot))
	for _, val := range activeVals {
		keys = append(keys, currentState.ValidatorRegistry[val].Pubkey)
	}
	assign, err := validatorServer.CommitteeAssignment(ctx, &pb.CommitteeAssignmentsRequest{
		PublicKeys: keys,
		EpochStart: currentState.Slot,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Info(assign)
}
