package debug

import (
	"bytes"
	"context"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pbrpc "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
)

func TestServer_GetBlock(t *testing.T) {
	db, _ := dbTest.SetupDB(t)
	ctx := context.Background()
	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{
		Slot: 100,
	}}
	if err := db.SaveBlock(ctx, b); err != nil {
		t.Fatal(err)
	}
	blockRoot, err := stateutil.BlockRoot(b.Block)
	if err != nil {
		t.Fatal(err)
	}
	bs := &Server{
		BeaconDB: db,
	}
	res, err := bs.GetBlock(ctx, &pbrpc.BlockRequest{
		BlockRoot: blockRoot[:],
	})
	if err != nil {
		t.Fatal(err)
	}
	wanted, err := b.MarshalSSZ()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(wanted, res.Encoded) {
		t.Errorf("Wanted %v, received %v", wanted, res.Encoded)
	}

	// Checking for nil block.
	blockRoot = [32]byte{}
	res, err = bs.GetBlock(ctx, &pbrpc.BlockRequest{
		BlockRoot: blockRoot[:],
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal([]byte{}, res.Encoded) {
		t.Errorf("Wanted empty, received %v", res.Encoded)
	}
}
