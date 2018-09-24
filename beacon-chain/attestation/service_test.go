package attestation

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/database"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestStartStop(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()

	config := &database.DBConfig{DataDir: "", Name: "", InMemory: true}
	db, err := database.NewDB(config)
	if err != nil {
		t.Fatalf("could not setup beaconDB: %v", err)
	}

	AttestHandler, err := NewHandler(db.DB())
	if err != nil {
		t.Fatalf("could not get attestation handler: %v", err)
	}

	cfg := &Config{
		Handler: AttestHandler,
	}

	attestService := NewAttestService(ctx, cfg)

	attestService.IncomingAttestationFeed()
	attestService.ContainsAttestation([]byte{}, [32]byte{})

	// Test the start function.
	attestService.Start()

	// Test the stop function.
	if err := attestService.Stop(); err != nil {
		t.Fatalf("unable to attest chain service: %v", err)
	}

	// The context should have been canceled.
	if attestService.ctx.Err() == nil {
		t.Error("context was not canceled")
	}

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

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

	attestationService := NewAttestService(context.Background(), cfg)

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

func TestContainsAttestations(t *testing.T) {
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

	attestationService := NewAttestService(context.Background(), cfg)

	attestation := types.NewAttestation(&pb.AggregatedAttestation{
		Slot:             0,
		ShardId:          0,
		AttesterBitfield: []byte{7}, // 0000 0111
	})
	if err := attestationService.handler.saveAttestation(attestation); err != nil {
		t.Fatalf("can not save attestation %v", err)
	}

	// Check if attestation exists for atteser bitfield 0000 0100
	exists, err := attestationService.ContainsAttestation([]byte{4}, attestation.Key())
	if err != nil {
		t.Fatalf("can not call ContainsAttestation %v", err)
	}
	if !exists {
		t.Error("Attestation should have existed")
	}

	// Check if attestation exists for atteser bitfield 0000 1000
	exists, err = attestationService.ContainsAttestation([]byte{8}, attestation.Key())
	if err != nil {
		t.Fatalf("can not call ContainsAttestation %v", err)
	}
	if exists {
		t.Error("Attestation shouldn't have existed")
	}
}
