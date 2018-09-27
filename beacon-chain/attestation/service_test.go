package attestation

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func setupService(t *testing.T) *Service {
	ctx := context.Background()

	config := db.Config{Path: "", Name: "", InMemory: true}
	db, err := db.NewDB(config)
	if err != nil {
		t.Fatalf("could not setup beaconDB: %v", err)
	}

	cfg := &Config{
		BeaconDB: db,
	}

	return NewAttestationService(ctx, cfg)
}

func TestIncomingAttestations(t *testing.T) {
	hook := logTest.NewGlobal()
	service := setupService(t)

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
