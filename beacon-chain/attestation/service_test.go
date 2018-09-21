package attestation

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	"github.com/prysmaticlabs/prysm/shared/database"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestIncomingAttestations(t *testing.T) {
	hook := logTest.NewGlobal()
	config := &database.DBConfig{DataDir: "", Name: "", InMemory: true}
	db, err := database.NewDB(config)
	if err != nil {
		t.Fatalf("could not setup beaconDB: %v", err)
	}

	attestationHandler, err := NewHandler(db.DB())
	if err != nil {
		t.Fatalf("could not register blockchain service: %v", err)
	}

	cfg := &Config{
		ReceiveAttestationBuf: 0,
		Handler:               attestationHandler,
	}

	attestationService := NewService(context.Background(), cfg)

	exitRoutine := make(chan bool)
	go func() {
		attestationService.aggregateAttestations()
		<-exitRoutine
	}()

	attestationService.incomingChan <- types.NewAttestation(nil)
	attestationService.cancel()
	exitRoutine <- true

	testutil.AssertLogsContain(t, hook, "Forwarding aggregated attestation")
}
