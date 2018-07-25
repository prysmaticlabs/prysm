package blockchain

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/beacon-chain/database"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestStartStop(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	tmp := fmt.Sprintf("%s/beacontest", os.TempDir())
	config := &database.BeaconDBConfig{DataDir: tmp, Name: "beacontestdata", InMemory: false}
	db, err := database.NewBeaconDB(config)
	if err != nil {
		t.Fatalf("could not setup beaconDB: %v", err)

	}
	db.Start()
	endpoint := "ws://127.0.0.1"
	web3Service, err := powchain.NewWeb3Service(ctx, &powchain.Web3ServiceConfig{endpoint, "", common.Address{}})
	if err != nil {
		t.Fatalf("unable to set up web3 service: %v", err)
	}
	chainService, err := NewChainService(ctx, db, web3Service)
	if err != nil {
		t.Fatalf("unable to setup chain service: %v", err)
	}

	chainService.Start()

	if err := chainService.Stop(); err != nil {
		t.Fatalf("unable to stop chain service: %v", err)
	}

	msg := hook.AllEntries()[0].Message
	want := "Starting beaconDB service"
	if msg != want {
		t.Errorf("incorrect log, expected %s, got %s", want, msg)
	}

	msg = hook.AllEntries()[1].Message
	want = "Starting service"
	if msg != want {
		t.Errorf("incorrect log, expected %s, got %s", want, msg)
	}

	msg = hook.AllEntries()[2].Message
	want = "No chainstate found on disk, initializing beacon from genesis"
	if msg != want {
		t.Errorf("incorrect log, expected %s, got %s", want, msg)
	}

	msg = hook.AllEntries()[3].Message
	want = "Stopping service"
	if msg != want {
		t.Errorf("incorrect log, expected %s, got %s", want, msg)
	}

	// The context should have been canceled.
	if chainService.ctx.Err() == nil {
		t.Error("context was not canceled")
	}
	hook.Reset()
}
