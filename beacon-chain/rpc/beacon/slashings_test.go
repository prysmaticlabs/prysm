package beacon

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/slashings"
)

func TestServer_SubmitProposerSlashing(t *testing.T) {
	ctx := context.Background()
	bs := &Server{
		SlashingsPool: slashings.NewPool(),
	}
	wanted := &ethpb.SubmitSlashingResponse{
		SlashedIndices: []uint64{0, 1, 2},
	}
	res, err := bs.SubmitProposerSlashing(ctx, &ethpb.ProposerSlashing{})
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(wanted, res) {
		t.Errorf("Wanted %v, received %v", wanted, res)
	}
}

func TestServer_SubmitAttesterSlashing(t *testing.T) {
	ctx := context.Background()
	bs := &Server{
		SlashingsPool: slashings.NewPool(),
	}
	wanted := &ethpb.SubmitSlashingResponse{
		SlashedIndices: []uint64{0, 1, 2},
	}
	res, err := bs.SubmitAttesterSlashing(ctx, &ethpb.AttesterSlashing{})
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(wanted, res) {
		t.Errorf("Wanted %v, received %v", wanted, res)
	}
}
