package proposals

import (
	"context"
	"crypto/rand"
	"reflect"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	testDB "github.com/prysmaticlabs/prysm/slasher/db/testing"
)

func signedBlockHeader(slot uint64, proposerIdx uint64) (*ethpb.SignedBeaconBlockHeader, error) {
	sig, err := genRandomSig()
	if err != nil {
		return nil, err
	}
	root := [32]byte{1, 2, 3}
	return &ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			//ProposerIndex proposerIndex,
			Slot:       slot,
			ParentRoot: root[:],
			StateRoot:  root[:],
			BodyRoot:   root[:],
		},
		Signature: sig,
	}, nil
}

func genRandomSig() (blk []byte, err error) {
	blk = make([]byte, 96)
	_, err = rand.Read(blk)
	return
}

func startSlot(epoch uint64) uint64 {
	return epoch * params.BeaconConfig().SlotsPerEpoch
}

func TestProposalsDetector_DetectSlashingsForBlockHeaders(t *testing.T) {
	type testStruct struct {
		name        string
		blk         *ethpb.SignedBeaconBlockHeader
		incomingBlk *ethpb.SignedBeaconBlockHeader
		slashing    *ethpb.ProposerSlashing
	}
	blk1epoch0, err := signedBlockHeader(startSlot(0), 0)
	if err != nil {
		t.Fatal(err)
	}
	blk2epoch0, err := signedBlockHeader(startSlot(0)+1, 0)
	if err != nil {
		t.Fatal(err)
	}
	blk1epoch1, err := signedBlockHeader(startSlot(1), 0)
	if err != nil {
		t.Fatal(err)
	}
	//blk1epoch3, err := signedBlockHeader(startSlot(3), 0)
	//if err != nil {
	//	t.Fatal(err)
	//}
	tests := []testStruct{
		{
			name:        "same block sig dont slash",
			blk:         blk1epoch0,
			incomingBlk: blk1epoch0,
			slashing:    nil,
		},
		{
			name:        "block from different epoch dont slash",
			blk:         blk1epoch0,
			incomingBlk: blk1epoch1,
			slashing:    nil,
		},
		{
			name:        "different sig from same epoch slash",
			blk:         blk1epoch0,
			incomingBlk: blk2epoch0,
			slashing:    &ethpb.ProposerSlashing{ProposerIndex: 0, Header_1: blk2epoch0, Header_2: blk1epoch0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testDB.SetupSlasherDB(t, false)
			defer testDB.TeardownSlasherDB(t, db)
			ctx := context.Background()

			sd := &ProposeDetector{
				slasherDB: db,
			}

			if err := sd.slasherDB.SaveBlockHeader(ctx, 0, tt.blk); err != nil {
				t.Fatal(err)
			}

			res, err := sd.DetectDoublePropose(ctx, tt.incomingBlk)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(res, tt.slashing) {
				t.Errorf("Wanted: %v, received %v", tt.slashing, res)
			}

		})
	}
}
