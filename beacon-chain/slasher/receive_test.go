package slasher

import (
	"context"
	"fmt"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestSlasher_receiveAttestations(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx, cancel := context.WithCancel(context.Background())
	s := &Service{
		serviceCfg: &ServiceConfig{
			IndexedAttsFeed: new(event.Feed),
		},
		indexedAttsChan: make(chan *ethpb.IndexedAttestation, 0),
	}
	exitChan := make(chan struct{})
	go func() {
		s.receiveAttestations(ctx)
		exitChan <- struct{}{}
	}()
	firstIndices := []uint64{1, 2, 3}
	secondIndices := []uint64{4, 5, 6}
	s.indexedAttsChan <- &ethpb.IndexedAttestation{
		AttestingIndices: firstIndices,
	}
	s.indexedAttsChan <- &ethpb.IndexedAttestation{
		AttestingIndices: secondIndices,
	}
	cancel()
	<-exitChan
	require.LogsContain(t, hook, fmt.Sprintf("Got attestation with indices %v", firstIndices))
	require.LogsContain(t, hook, fmt.Sprintf("Got attestation with indices %v", secondIndices))
}
