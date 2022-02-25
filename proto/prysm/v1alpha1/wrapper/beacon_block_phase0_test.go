package wrapper_test

import (
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestPhase0SignedBeaconBlock_Header(t *testing.T) {
	root := bytesutil.PadTo([]byte("root"), 32)
	signature := bytesutil.PadTo([]byte("sig"), 96)
	body := &ethpb.BeaconBlockBody{}
	body = util.HydrateBeaconBlockBody(body)
	bodyRoot, err := body.HashTreeRoot()
	require.NoError(t, err)
	block := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot:          1,
			ProposerIndex: 1,
			ParentRoot:    root,
			StateRoot:     root,
			Body:          body,
		},
		Signature: signature,
	}
	wrapped := wrapper.WrappedPhase0SignedBeaconBlock(block)

	header, err := wrapped.Header()
	require.NoError(t, err)
	assert.Equal(t, types.ValidatorIndex(1), header.Header.ProposerIndex)
	assert.Equal(t, types.Slot(1), header.Header.Slot)
	assert.DeepEqual(t, bodyRoot[:], header.Header.BodyRoot)
	assert.DeepEqual(t, root, header.Header.StateRoot)
	assert.DeepEqual(t, root, header.Header.ParentRoot)
	assert.DeepEqual(t, signature, header.Signature)
}
