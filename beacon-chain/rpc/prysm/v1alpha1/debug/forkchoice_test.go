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

func TestServer_GetForkChoice(t *testing.T) {
	store := &protoarray.Store{}
	bs := &Server{HeadFetcher: &mock.ChainService{ForkChoiceStore: store}}
	res, err := bs.GetProtoArrayForkChoice(context.Background(), &empty.Empty{})
	require.NoError(t, err)
	assert.Equal(t, store.PruneThreshold(), res.PruneThreshold, "Did not get wanted prune threshold")
	assert.Equal(t, store.JustifiedEpoch(), res.JustifiedEpoch, "Did not get wanted justified epoch")
	assert.Equal(t, store.FinalizedEpoch(), res.FinalizedEpoch, "Did not get wanted finalized epoch")
}
