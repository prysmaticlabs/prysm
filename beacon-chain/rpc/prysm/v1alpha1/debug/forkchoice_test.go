package debug

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestServer_GetForkChoice_ProtoArray(t *testing.T) {
	store := protoarray.New()
	bs := &Server{ForkFetcher: &mock.ChainService{ForkChoiceStore: store}}
	res, err := bs.GetForkChoice(context.Background(), &empty.Empty{})
	require.NoError(t, err)
	assert.Equal(t, store.JustifiedCheckpoint().Epoch, res.JustifiedEpoch, "Did not get wanted justified epoch")
	assert.Equal(t, store.FinalizedCheckpoint().Epoch, res.FinalizedEpoch, "Did not get wanted finalized epoch")
}
