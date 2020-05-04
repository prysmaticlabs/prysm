package proposals

import (
	"context"
	"reflect"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	testDB "github.com/prysmaticlabs/prysm/slasher/db/testing"
	"github.com/prysmaticlabs/prysm/slasher/detection/proposals/iface"
	testDetect "github.com/prysmaticlabs/prysm/slasher/detection/testing"
)

var _ = iface.ProposalsDetector(&ProposeDetector{})

func TestProposalsDetector_DetectSlashingsForBlockHeaders(t *testing.T) {
	type testStruct struct {
		name        string
		blk         *ethpb.SignedBeaconBlockHeader
		incomingBlk *ethpb.SignedBeaconBlockHeader
		slashing    *ethpb.ProposerSlashing
	}
	blk1slot0, err := testDetect.SignedBlockHeader(testDetect.StartSlot(0), 0)
	if err != nil {
		t.Fatal(err)
	}
	blk2slot0, err := testDetect.SignedBlockHeader(testDetect.StartSlot(0), 0)
	if err != nil {
		t.Fatal(err)
	}
	blk1slot1, err := testDetect.SignedBlockHeader(testDetect.StartSlot(0)+1, 0)
	if err != nil {
		t.Fatal(err)
	}
	blk1epoch1, err := testDetect.SignedBlockHeader(testDetect.StartSlot(1), 0)
	if err != nil {
		t.Fatal(err)
	}
	tests := []testStruct{
		{
			name:        "same block sig dont slash",
			blk:         blk1slot0,
			incomingBlk: blk1slot0,
			slashing:    nil,
		},
		{
			name:        "block from different epoch dont slash",
			blk:         blk1slot0,
			incomingBlk: blk1epoch1,
			slashing:    nil,
		},
		{
			name:        "different sig from different slot dont slash",
			blk:         blk1slot0,
			incomingBlk: blk1slot1,
			slashing:    nil,
		},
		{
			name:        "different sig from same slot slash",
			blk:         blk1slot0,
			incomingBlk: blk2slot0,
			slashing:    &ethpb.ProposerSlashing{Header_1: blk2slot0, Header_2: blk1slot0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testDB.SetupSlasherDB(t, false)
			ctx := context.Background()

			sd := &ProposeDetector{
				slasherDB: db,
			}

			if err := sd.slasherDB.SaveBlockHeader(ctx, tt.blk); err != nil {
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
