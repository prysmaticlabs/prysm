package kv

import (
	"context"
	"flag"
	"reflect"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/urfave/cli/v2"
)

func TestNilDBHistoryBlkHdr(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(&app, set, nil))
	ctx := context.Background()

	slot := uint64(1)
	validatorID := uint64(1)

	hasBlockHeader := db.HasBlockHeader(ctx, slot, validatorID)
	if hasBlockHeader {
		t.Fatal("HasBlockHeader should return false")
	}

	bPrime, err := db.BlockHeaders(ctx, slot, validatorID)
	if err != nil {
		t.Fatalf("failed to get block: %v", err)
	}
	if bPrime != nil {
		t.Fatalf("get should return nil for a non existent key")
	}
}

func TestSaveHistoryBlkHdr(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(&app, set, nil))
	ctx := context.Background()

	tests := []struct {
		bh *ethpb.SignedBeaconBlockHeader
	}{
		{
			bh: &ethpb.SignedBeaconBlockHeader{Signature: bytesutil.PadTo([]byte("let me in"), 96), Header: &ethpb.BeaconBlockHeader{Slot: 0, ProposerIndex: 0}},
		},
		{
			bh: &ethpb.SignedBeaconBlockHeader{Signature: bytesutil.PadTo([]byte("let me in 2nd"), 96), Header: &ethpb.BeaconBlockHeader{Slot: 0, ProposerIndex: 1}},
		},
		{
			bh: &ethpb.SignedBeaconBlockHeader{Signature: bytesutil.PadTo([]byte("let me in 3rd"), 96), Header: &ethpb.BeaconBlockHeader{Slot: params.BeaconConfig().SlotsPerEpoch + 1, ProposerIndex: 0}},
		},
		{
			bh: &ethpb.SignedBeaconBlockHeader{Signature: bytesutil.PadTo([]byte("let me in 3rd"), 96), Header: &ethpb.BeaconBlockHeader{Slot: 1, ProposerIndex: 0}},
		},
	}

	for _, tt := range tests {
		err := db.SaveBlockHeader(ctx, tt.bh)
		if err != nil {
			t.Fatalf("save block failed: %v", err)
		}

		bha, err := db.BlockHeaders(ctx, tt.bh.Header.Slot, tt.bh.Header.ProposerIndex)
		if err != nil {
			t.Fatalf("failed to get block: %v", err)
		}

		if bha == nil || !reflect.DeepEqual(bha[0], tt.bh) {
			t.Fatalf("get should return bh: %v", bha)
		}
	}

}

func TestDeleteHistoryBlkHdr(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(&app, set, nil))
	ctx := context.Background()

	tests := []struct {
		bh *ethpb.SignedBeaconBlockHeader
	}{
		{
			bh: &ethpb.SignedBeaconBlockHeader{Signature: bytesutil.PadTo([]byte("let me in"), 96), Header: &ethpb.BeaconBlockHeader{Slot: 0, ProposerIndex: 0}},
		},
		{
			bh: &ethpb.SignedBeaconBlockHeader{Signature: bytesutil.PadTo([]byte("let me in 2nd"), 96), Header: &ethpb.BeaconBlockHeader{Slot: 0, ProposerIndex: 1}},
		},
		{
			bh: &ethpb.SignedBeaconBlockHeader{Signature: bytesutil.PadTo([]byte("let me in 3rd"), 96), Header: &ethpb.BeaconBlockHeader{Slot: params.BeaconConfig().SlotsPerEpoch + 1, ProposerIndex: 0}},
		},
	}
	for _, tt := range tests {

		err := db.SaveBlockHeader(ctx, tt.bh)
		if err != nil {
			t.Fatalf("save block failed: %v", err)
		}
	}

	for _, tt := range tests {
		bha, err := db.BlockHeaders(ctx, tt.bh.Header.Slot, tt.bh.Header.ProposerIndex)
		if err != nil {
			t.Fatalf("failed to get block: %v", err)
		}

		if bha == nil || !reflect.DeepEqual(bha[0], tt.bh) {
			t.Fatalf("get should return bh: %v", bha)
		}
		err = db.DeleteBlockHeader(ctx, tt.bh)
		if err != nil {
			t.Fatalf("save block failed: %v", err)
		}
		bh, err := db.BlockHeaders(ctx, tt.bh.Header.Slot, tt.bh.Header.ProposerIndex)

		if err != nil {
			t.Fatal(err)
		}
		if bh != nil {
			t.Errorf("Expected block to have been deleted, received: %v", bh)
		}

	}

}

func TestHasHistoryBlkHdr(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(&app, set, nil))
	ctx := context.Background()

	tests := []struct {
		bh *ethpb.SignedBeaconBlockHeader
	}{
		{
			bh: &ethpb.SignedBeaconBlockHeader{Signature: bytesutil.PadTo([]byte("let me in"), 96), Header: &ethpb.BeaconBlockHeader{Slot: 0, ProposerIndex: 0}},
		},
		{
			bh: &ethpb.SignedBeaconBlockHeader{Signature: bytesutil.PadTo([]byte("let me in 2nd"), 96), Header: &ethpb.BeaconBlockHeader{Slot: 0, ProposerIndex: 1}},
		},
		{
			bh: &ethpb.SignedBeaconBlockHeader{Signature: bytesutil.PadTo([]byte("let me in 3rd"), 96), Header: &ethpb.BeaconBlockHeader{Slot: params.BeaconConfig().SlotsPerEpoch + 1, ProposerIndex: 0}},
		},
		{
			bh: &ethpb.SignedBeaconBlockHeader{Signature: bytesutil.PadTo([]byte("let me in 4th"), 96), Header: &ethpb.BeaconBlockHeader{Slot: 1, ProposerIndex: 0}},
		},
	}
	for _, tt := range tests {

		found := db.HasBlockHeader(ctx, tt.bh.Header.Slot, tt.bh.Header.ProposerIndex)
		if found {
			t.Fatal("has block header should return false for block headers that are not in db")
		}
		err := db.SaveBlockHeader(ctx, tt.bh)
		if err != nil {
			t.Fatalf("save block failed: %v", err)
		}
	}
	for _, tt := range tests {
		err := db.SaveBlockHeader(ctx, tt.bh)
		if err != nil {
			t.Fatalf("save block failed: %v", err)
		}

		found := db.HasBlockHeader(ctx, tt.bh.Header.Slot, tt.bh.Header.ProposerIndex)

		if !found {
			t.Fatal("has block header should return true")
		}
	}
}

func TestPruneHistoryBlkHdr(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(&app, set, nil))
	ctx := context.Background()

	tests := []struct {
		bh *ethpb.SignedBeaconBlockHeader
	}{
		{
			bh: &ethpb.SignedBeaconBlockHeader{Signature: bytesutil.PadTo([]byte("let me in"), 96), Header: &ethpb.BeaconBlockHeader{Slot: 0, ProposerIndex: 0}},
		},
		{
			bh: &ethpb.SignedBeaconBlockHeader{Signature: bytesutil.PadTo([]byte("let me in 2nd"), 96), Header: &ethpb.BeaconBlockHeader{Slot: 0, ProposerIndex: 1}},
		},
		{
			bh: &ethpb.SignedBeaconBlockHeader{Signature: bytesutil.PadTo([]byte("let me in 3rd"), 96), Header: &ethpb.BeaconBlockHeader{Slot: params.BeaconConfig().SlotsPerEpoch + 1, ProposerIndex: 0}},
		},
		{
			bh: &ethpb.SignedBeaconBlockHeader{Signature: bytesutil.PadTo([]byte("let me in 4th"), 96), Header: &ethpb.BeaconBlockHeader{Slot: params.BeaconConfig().SlotsPerEpoch*2 + 1, ProposerIndex: 0}},
		},
		{
			bh: &ethpb.SignedBeaconBlockHeader{Signature: bytesutil.PadTo([]byte("let me in 5th"), 96), Header: &ethpb.BeaconBlockHeader{Slot: params.BeaconConfig().SlotsPerEpoch*3 + 1, ProposerIndex: 0}},
		},
	}

	for _, tt := range tests {
		err := db.SaveBlockHeader(ctx, tt.bh)
		if err != nil {
			t.Fatalf("save block header failed: %v", err)
		}

		bha, err := db.BlockHeaders(ctx, tt.bh.Header.Slot, tt.bh.Header.ProposerIndex)
		if err != nil {
			t.Fatalf("failed to get block header: %v", err)
		}

		if bha == nil || !reflect.DeepEqual(bha[0], tt.bh) {
			t.Fatalf("get should return bh: %v", bha)
		}
	}
	currentEpoch := uint64(3)
	historyToKeep := uint64(2)
	err := db.PruneBlockHistory(ctx, currentEpoch, historyToKeep)
	if err != nil {
		t.Fatalf("failed to prune: %v", err)
	}

	for _, tt := range tests {
		bha, err := db.BlockHeaders(ctx, tt.bh.Header.Slot, tt.bh.Header.ProposerIndex)
		if err != nil {
			t.Fatalf("failed to get block header: %v", err)
		}
		if helpers.SlotToEpoch(tt.bh.Header.Slot) >= currentEpoch-historyToKeep {
			if bha == nil || !reflect.DeepEqual(bha[0], tt.bh) {
				t.Fatalf("get should return bh: %v", bha)
			}
		} else {
			if bha != nil {
				t.Fatalf("block header should have been pruned: %v", bha)
			}
		}
	}
}
