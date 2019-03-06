package blockchain

import (
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared/hashutil"

	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestGenerateState_OK(t *testing.T) {
	beaconState, err := state.GenesisBeaconState(nil, 0, nil)
	if err != nil {
		t.Fatalf("Cannot create genesis beacon state: %v", err)
	}
	stateRoot, err := hashutil.HashProto(beaconState)
	if err != nil {
		t.Fatalf("Could not tree hash state: %v", err)
	}

	genesis := b.NewGenesisBlock(stateRoot[:])
	genesisRoot, err := hashutil.HashProto(genesis)
	if err != nil {
		t.Fatalf("Could not get genesis block root: %v", err)
	}

	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	chainService := setupBeaconChain(t, false, db, true)
	unixTime := uint64(time.Now().Unix())
	deposits, _ := setupInitialDeposits(t, 100)
	if err := db.InitializeState(unixTime, deposits); err != nil {
		t.Fatalf("Could not initialize beacon state to disk: %v", err)
	}

	stateRoot, err := hashutil.HashProto(tt.state)
	if err != nil {
		t.Fatalf("Could not tree hash state: %v", err)
	}
	block := &pb.BeaconBlock{
		Slot:             tt.blockSlot,
		StateRootHash32:  stateRoot[:],
		ParentRootHash32: genesisRoot[:],
		Eth1Data: &pb.Eth1Data{
			DepositRootHash32: []byte("a"),
			BlockHash32:       []byte("b"),
		},
	}
	if err := chainService.beaconDB.SaveBlock(block); err != nil {
		t.Fatal(err)
	}
}
