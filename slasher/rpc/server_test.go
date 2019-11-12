package rpc

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/slasher/db"
)

func TestServer_IsSlashableBlock(t *testing.T) {
	dbs := db.SetupSlasherDB(t)
	defer db.TeardownSlasherDB(t, dbs)
	ctx := context.Background()
	slasherServer := &Server{
		SlasherDB: dbs,
	}
	psr := &ethpb.ProposerSlashingRequest{
		BlockHeader: &ethpb.BeaconBlockHeader{
			Slot:      1,
			StateRoot: []byte("A"),
		},
		ValidatorIndex: 1,
	}
	psr2 := &ethpb.ProposerSlashingRequest{
		BlockHeader: &ethpb.BeaconBlockHeader{
			Slot:      1,
			StateRoot: []byte("B"),
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
		t.Errorf("Should return 1 slashaing proof: %v", sr)
	}
	if !proto.Equal(sr.ProposerSlashing[0], want) {
		t.Errorf("wanted slashing proof: %v got: %v", want, sr.ProposerSlashing[0])

	}

}

func TestServer_IsNotSlashableBlock(t *testing.T) {
	dbs := db.SetupSlasherDB(t)
	defer db.TeardownSlasherDB(t, dbs)

	slasherServer := &Server{
		SlasherDB: dbs,
	}
	psr := &ethpb.ProposerSlashingRequest{
		BlockHeader: &ethpb.BeaconBlockHeader{
			Slot:      1,
			StateRoot: []byte("A"),
		},
		ValidatorIndex: 1,
	}
	psr2 := &ethpb.ProposerSlashingRequest{
		BlockHeader: &ethpb.BeaconBlockHeader{
			Slot:      65,
			StateRoot: []byte("B"),
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
		t.Errorf("Should return 0 slashaing proof: %v", sr)
	}

}

func TestServer_DoubleBlock(t *testing.T) {
	dbs := db.SetupSlasherDB(t)
	defer db.TeardownSlasherDB(t, dbs)
	ctx := context.Background()
	slasherServer := &Server{
		ctx:       ctx,
		SlasherDB: dbs,
	}
	psr := &ethpb.ProposerSlashingRequest{
		BlockHeader: &ethpb.BeaconBlockHeader{
			Slot:      1,
			StateRoot: []byte("A"),
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
		t.Errorf("Should return 0 slashaing proof: %v", sr)
	}

}

func TestServer_SameSlotSlashable(t *testing.T) {
	dbs := db.SetupSlasherDB(t)
	defer db.TeardownSlasherDB(t, dbs)
	ctx := context.Background()
	slasherServer := &Server{
		ctx:       ctx,
		SlasherDB: dbs,
	}
	psr := &ethpb.ProposerSlashingRequest{
		BlockHeader: &ethpb.BeaconBlockHeader{
			Slot:      1,
			StateRoot: []byte("A"),
		},
		ValidatorIndex: 1,
	}
	psr2 := &ethpb.ProposerSlashingRequest{
		BlockHeader: &ethpb.BeaconBlockHeader{
			Slot:      1,
			StateRoot: []byte("B"),
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
		t.Errorf("Should return 1 slashaing proof: %v", sr)
	}
	if !proto.Equal(sr.ProposerSlashing[0], want) {
		t.Errorf("wanted slashing proof: %v got: %v", want, sr.ProposerSlashing[0])

	}
}
