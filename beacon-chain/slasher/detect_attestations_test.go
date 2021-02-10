package slasher

import (
	"context"
	"testing"

	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestService_processQueuedAttestations(t *testing.T) {
	hook := logTest.NewGlobal()
	s := &Service{
		params: DefaultParams(),
		attestationQueue: []*slashertypes.CompactAttestation{
			{
				AttestingIndices: []uint64{0, 1},
				Source:           0,
				Target:           1,
			},
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	tickerChan := make(chan uint64)
	exitChan := make(chan struct{})
	go func() {
		s.processQueuedAttestations(ctx, tickerChan)
		exitChan <- struct{}{}
	}()

	// Send a value over the ticker.
	tickerChan <- 0
	cancel()
	<-exitChan
	assert.LogsContain(t, hook, "Epoch reached, processing queued")
}
