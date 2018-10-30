package attestation

import (
	"context"
	"testing"

	btestutil "github.com/prysmaticlabs/prysm/beacon-chain/testutil"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

func TestIncomingAttestations(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB := btestutil.SetupDB(t)
	defer btestutil.TeardownDB(t, beaconDB)
	service := NewAttestationService(context.Background(), &Config{BeaconDB: beaconDB})

	exitRoutine := make(chan bool)
	go func() {
		service.aggregateAttestations()
		<-exitRoutine
	}()

	service.incomingChan <- types.NewAttestation(nil)
	service.cancel()
	exitRoutine <- true

	testutil.AssertLogsContain(t, hook, "Forwarding aggregated attestation")
}
