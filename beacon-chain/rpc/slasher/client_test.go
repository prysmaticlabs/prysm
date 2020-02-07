package slasher

import (
	"context"
	"strings"
	"testing"
	"time"

	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/slashings"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func init() {
	// Use minimal config to reduce test setup time.
	params.OverrideBeaconConfig(params.MinimalSpecConfig())
	params.BeaconConfig().SecondsPerSlot = 1
}

func TestClient_SlashingPoolFeeder_ContextCanceled(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	s := &beaconstate.BeaconState{}

	slasherClient := &Client{
		HeadFetcher: &mock.ChainService{State: s},
	}

	wanted := "Stream context canceled"
	var err error
	go func() {
		err = slasherClient.SlashingPoolFeeder(ctx)
	}()
	cancel()
	time.Sleep(time.Millisecond)
	if !strings.Contains(err.Error(), wanted) {
		t.Error("Did not receive wanted error")
	}
}

func TestClient_SlashingPoolFeeder_NoSlasher(t *testing.T) {
	ctx := context.Background()
	s := &beaconstate.BeaconState{}

	slasherClient := &Client{
		HeadFetcher: &mock.ChainService{State: s},
	}

	wanted := "Slasher server has not been started"
	var err error
	go func() {
		err = slasherClient.SlashingPoolFeeder(ctx)
	}()
	time.Sleep(time.Second * 9)
	if !strings.Contains(err.Error(), wanted) {
		t.Error("Did not receive wanted error")
	}
}

func TestClient_UpdatePool(t *testing.T) {
	ctx := context.Background()
	s := &beaconstate.BeaconState{}

	slasherClient := &fakeSlasher{}

	client := &Client{
		HeadFetcher:     &mock.ChainService{State: s},
		SlashingPool:    slashings.NewPool(),
		SlasherClient:   slasherClient,
		ShouldBroadcast: false,
	}

	if err := client.updatePool(ctx); err != nil {
		t.Fatal(err)
	}

	if !slasherClient.ProposerSlashingsCalled {
		t.Fatal("fuck")
	}
	if !slasherClient.AttesterSlashingsCalled {
		t.Fatal("fuck")
	}
}
