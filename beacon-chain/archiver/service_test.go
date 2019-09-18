package archiver

import (
	"context"
	"fmt"
	"testing"

	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestArchiverService_ReceivesNewChainHead(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx, cancel := context.WithCancel(context.Background())
	svc := &Service{
		ctx:             ctx,
		cancel:          cancel,
		newHeadRootChan: make(chan [32]byte, 0),
		newHeadNotifier: &mock.ChainService{},
	}
	exitRoutine := make(chan bool)
	go func() {
		svc.run()
		<-exitRoutine
	}()

	svc.newHeadRootChan <- [32]byte{1, 2, 3}
	if err := svc.Stop(); err != nil {
		t.Fatal(err)
	}
	exitRoutine <- true

	// The context should have been canceled.
	if svc.ctx.Err() != context.Canceled {
		t.Error("context was not canceled")
	}
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("%#x", [32]byte{1, 2, 3}))
	testutil.AssertLogsContain(t, hook, "New chain head event")
}
