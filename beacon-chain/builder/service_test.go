package builder

import (
	"context"
	"encoding/hex"
	"testing"

	"github.com/prysmaticlabs/prysm/api/client/builder"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestMergeMockRoundtrip(t *testing.T) {
	c, err := builder.NewClient("http://localhost:28545")
	require.NoError(t, err)

	h := "a0513a503d5bd6e89a144c3268e5b7e9da9dbf63df125a360e3950a7d0d67131"
	data, err := hex.DecodeString(h)
	require.NoError(t, err)
	ctx := context.Background()
	header, err := c.GetHeader(ctx, 1, bytesutil.ToBytes32(data), [48]byte{})
	require.NoError(t, err)
	t.Log(header.Message.Value)

	st, keys := util.DeterministicGenesisState(t, 64)
	b, err := util.GenerateFullBlock(st, keys, util.DefaultBlockGenConfig(), 1)
	require.NoError(t, err)

	sb := HydrateSignedBlindedBeaconBlockBellatrix(&ethpb.SignedBlindedBeaconBlockBellatrix{
		Signature: keys[0].Sign([]byte("hello")).Marshal(),
		Block: &ethpb.BlindedBeaconBlockBellatrix{
			Slot:          b.Block.Slot,
			ParentRoot:    b.Block.ParentRoot,
			StateRoot:     b.Block.StateRoot,
			ProposerIndex: b.Block.ProposerIndex,
			Body: &ethpb.BlindedBeaconBlockBodyBellatrix{
				Attestations:           b.Block.Body.Attestations,
				RandaoReveal:           b.Block.Body.RandaoReveal,
				Deposits:               b.Block.Body.Deposits,
				VoluntaryExits:         b.Block.Body.VoluntaryExits,
				ProposerSlashings:      b.Block.Body.ProposerSlashings,
				AttesterSlashings:      b.Block.Body.AttesterSlashings,
				Graffiti:               b.Block.Body.Graffiti,
				ExecutionPayloadHeader: header.Message.Header,
			},
		},
	})
	if _, err := c.SubmitBlindedBlock(ctx, sb); err != nil {
		t.Fatal(err)
	}
}
