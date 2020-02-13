package rpc

import (
	"context"
	"flag"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	testDB "github.com/prysmaticlabs/prysm/slasher/db/testing"
	"github.com/prysmaticlabs/prysm/slasher/db/types"
	"github.com/urfave/cli"
)

func TestServer_IsSlashableBlock(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	c := cli.NewContext(app, set, nil)
	db := testDB.SetupSlasherDB(t, c)
	defer testDB.TeardownSlasherDB(t, db)
	ctx := context.Background()
	slasherServer := &Server{
		SlasherDB: db,
	}
	psr := &slashpb.ProposerSlashingRequest{
		BlockHeader: &ethpb.SignedBeaconBlockHeader{
			Header: &ethpb.BeaconBlockHeader{
				Slot:      1,
				StateRoot: []byte("A"),
			},
		},
		ValidatorIndex: 1,
	}
	psr2 := &slashpb.ProposerSlashingRequest{
		BlockHeader: &ethpb.SignedBeaconBlockHeader{
			Header: &ethpb.BeaconBlockHeader{
				Slot:      1,
				StateRoot: []byte("B"),
			},
		},
		ValidatorIndex: 1,
	}

	if _, err := slasherServer.IsSlashableBlock(ctx, psr); err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	sr, err := slasherServer.IsSlashableBlock(ctx, psr2)
	if err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	want := &ethpb.ProposerSlashing{
		ProposerIndex: psr.ValidatorIndex,
		Header_1:      psr2.BlockHeader,
		Header_2:      psr.BlockHeader,
	}

	if len(sr.ProposerSlashing) != 1 {
		t.Errorf("Should return 1 slashing proof: %v", sr)
	}
	if !proto.Equal(sr.ProposerSlashing[0], want) {
		t.Errorf("wanted slashing proof: %v got: %v", want, sr.ProposerSlashing[0])

	}

}

func TestServer_IsNotSlashableBlock(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	c := cli.NewContext(app, set, nil)
	db := testDB.SetupSlasherDB(t, c)
	defer testDB.TeardownSlasherDB(t, db)

	slasherServer := &Server{
		SlasherDB: db,
	}
	psr := &slashpb.ProposerSlashingRequest{
		BlockHeader: &ethpb.SignedBeaconBlockHeader{
			Header: &ethpb.BeaconBlockHeader{
				Slot:      1,
				StateRoot: []byte("A"),
			},
		},
		ValidatorIndex: 1,
	}
	psr2 := &slashpb.ProposerSlashingRequest{
		BlockHeader: &ethpb.SignedBeaconBlockHeader{
			Header: &ethpb.BeaconBlockHeader{
				Slot:      65,
				StateRoot: []byte("B"),
			},
		},
		ValidatorIndex: 1,
	}
	ctx := context.Background()

	if _, err := slasherServer.IsSlashableBlock(ctx, psr); err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	sr, err := slasherServer.IsSlashableBlock(ctx, psr2)
	if err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}

	if len(sr.ProposerSlashing) != 0 {
		t.Errorf("Should return 0 slashing proof: %v", sr)
	}

}

func TestServer_DoubleBlock(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	c := cli.NewContext(app, set, nil)
	db := testDB.SetupSlasherDB(t, c)
	defer testDB.TeardownSlasherDB(t, db)
	ctx := context.Background()
	slasherServer := &Server{
		ctx:       ctx,
		SlasherDB: db,
	}
	psr := &slashpb.ProposerSlashingRequest{
		BlockHeader: &ethpb.SignedBeaconBlockHeader{
			Header: &ethpb.BeaconBlockHeader{
				Slot:      1,
				StateRoot: []byte("A"),
			},
		},
		ValidatorIndex: 1,
	}

	if _, err := slasherServer.IsSlashableBlock(ctx, psr); err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	sr, err := slasherServer.IsSlashableBlock(ctx, psr)
	if err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}

	if len(sr.ProposerSlashing) != 0 {
		t.Errorf("Should return 0 slashing proof: %v", sr)
	}

}

func TestServer_SameSlotSlashable(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	c := cli.NewContext(app, set, nil)
	db := testDB.SetupSlasherDB(t, c)
	defer testDB.TeardownSlasherDB(t, db)
	ctx := context.Background()
	slasherServer := &Server{
		ctx:       ctx,
		SlasherDB: db,
	}
	psr := &slashpb.ProposerSlashingRequest{
		BlockHeader: &ethpb.SignedBeaconBlockHeader{
			Header: &ethpb.BeaconBlockHeader{
				Slot:      1,
				StateRoot: []byte("A"),
			},
		},
		ValidatorIndex: 1,
	}
	psr2 := &slashpb.ProposerSlashingRequest{
		BlockHeader: &ethpb.SignedBeaconBlockHeader{
			Header: &ethpb.BeaconBlockHeader{
				Slot:      1,
				StateRoot: []byte("B"),
			},
		},
		ValidatorIndex: 1,
	}
	want := &ethpb.ProposerSlashing{
		ProposerIndex: psr.ValidatorIndex,
		Header_1:      psr2.BlockHeader,
		Header_2:      psr.BlockHeader,
	}

	if _, err := slasherServer.IsSlashableBlock(ctx, psr); err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	sr, err := slasherServer.IsSlashableBlock(ctx, psr2)
	if err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}

	if len(sr.ProposerSlashing) != 1 {
		t.Errorf("Should return 1 slashing proof: %v", sr)
	}
	if !proto.Equal(sr.ProposerSlashing[0], want) {
		t.Errorf("wanted slashing proof: %v got: %v", want, sr.ProposerSlashing[0])

	}
	if err := slasherServer.SlasherDB.SaveProposerSlashing(types.Active, sr.ProposerSlashing[0]); err != nil {
		t.Errorf("Could not call db method: %v", err)
	}
	if sr, err = slasherServer.ProposerSlashings(ctx, &slashpb.SlashingStatusRequest{Status: slashpb.SlashingStatusRequest_Active}); err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	ar, err := slasherServer.AttesterSlashings(ctx, &slashpb.SlashingStatusRequest{Status: slashpb.SlashingStatusRequest_Active})
	if err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	if len(ar.AttesterSlashing) > 0 {
		t.Errorf("Attester slashings with status 'active' should not be present in testDB.")
	}
	emptySlashingResponse, err := slasherServer.ProposerSlashings(ctx, &slashpb.SlashingStatusRequest{Status: slashpb.SlashingStatusRequest_Included})
	if err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	if len(emptySlashingResponse.ProposerSlashing) > 0 {
		t.Error("Proposer slashings with status 'included' should not be present in db")
	}
	if !proto.Equal(sr.ProposerSlashing[0], want) {
		t.Errorf("Wanted slashing proof: %v got: %v", want, sr.ProposerSlashing[0])
	}
}
