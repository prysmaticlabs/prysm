package operation

import (
	"context"
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

func TestStop(t *testing.T) {
	hook := logTest.NewGlobal()
	opsService := NewOperationService(context.Background(), &Config{})

	if err := opsService.Stop(); err != nil {
		t.Fatalf("Unable to stop operation service: %v", err)
	}

	msg := hook.LastEntry().Message
	want := "Stopping service"
	if msg != want {
		t.Errorf("incorrect log, expected %s, got %s", want, msg)
	}

	// The context should have been canceled.
	if opsService.ctx.Err() == nil {
		t.Error("context was not canceled")
	}
	hook.Reset()
}

func TestIncomingDeposits_1stDeposit(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	service := NewOperationService(context.Background(), &Config{BeaconDB: beaconDB})

	exitRoutine := make(chan bool)
	go func() {
		service.saveOperations()
		<-exitRoutine
	}()
	deposit := &pb.Deposit{DepositData: []byte{'A'}}
	hash, err := hashutil.HashProto(deposit)
	if err != nil {
		t.Fatalf("Could not hash deposit proto: %v", err)
	}

	service.incomingDepositChan <- deposit
	service.cancel()
	exitRoutine <- true

	want := fmt.Sprintf("Deposit %#x saved in db", hash)
	testutil.AssertLogsContain(t, hook, want)
}

func TestIncomingDeposits_2ndDeposit(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	service := NewOperationService(context.Background(), &Config{BeaconDB: beaconDB})

	exitRoutine := make(chan bool)
	go func() {
		service.saveOperations()
		<-exitRoutine
	}()
	deposit := &pb.Deposit{DepositData: []byte{'A'}}
	hash, err := hashutil.HashProto(deposit)
	if err != nil {
		t.Fatalf("Could not hash deposit proto: %v", err)
	}

	service.incomingDepositChan <- deposit
	service.incomingDepositChan <- deposit
	service.cancel()
	exitRoutine <- true

	want := fmt.Sprintf("Received. skipping deposit #%x", hash)
	testutil.AssertLogsContain(t, hook, want)
}
