package attestation

import (
	"context"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	"github.com/prysmaticlabs/prysm/shared/database"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"testing"
)

func TestIncomingAttestations(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	config := &database.DBConfig{DataDir: "", Name: "", InMemory: true}
	db, err := database.NewDB(config)
	if err != nil {
		t.Fatalf("could not setup beaconDB: %v", err)
	}

	attestationHandler, err := NewAttestationHandler(db.DB())
	if err != nil {
		t.Fatalf("could not register blockchain service: %v", err)
	}

	cfg := &Config{
		receiveAttestationBuf: 0,
		handler:               attestationHandler,
	}

	attestationService := NewAttestationService(ctx, cfg)

	exitRoutine := make(chan bool)
	go func() {
		attestationService.attestationProcessing()
		<-exitRoutine
	}()

	attestationService.receiveChan <- types.NewAttestation(nil)
	attestationService.cancel()
	exitRoutine <- true
	testutil.AssertLogsContain(t, hook, "Relaying attestation")
}
