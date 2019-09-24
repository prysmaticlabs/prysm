package slasher

import (
	"context"
	"reflect"
	"testing"

	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	testdb "github.com/prysmaticlabs/prysm/slasher/db/testing"
)

func TestServer_IsSlashableBlock(t *testing.T) {
	dbs := testdb.SetupSlasherDB(t)

	defer testdb.TeardownSlasherDB(t, dbs)
	ctx := context.Background()
	slasherServer := &Server{
		ctx:       ctx,
		slasherDb: dbs,
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
	if !reflect.DeepEqual(sr.ProposerSlashing[0], want) {
		t.Errorf("wanted slashing proof: %v got: %v", want, sr.ProposerSlashing[0])

	}

}
