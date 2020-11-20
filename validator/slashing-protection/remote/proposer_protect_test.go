package remote

import (
	"context"
	"testing"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestRemoteProtector_IsSlashableBlock(t *testing.T) {
	s := &Service{slasherClient: mockSlasher{slashBlock: true}}
	blk := &eth.SignedBeaconBlock{
		Block: &eth.BeaconBlock{
			Slot:          0,
			ProposerIndex: 0,
			ParentRoot:    bytesutil.PadTo([]byte("parent"), 32),
			StateRoot:     bytesutil.PadTo([]byte("state"), 32),
			Body:          &eth.BeaconBlockBody{},
		},
	}
	ctx := context.Background()
	slashable, err := s.IsSlashableBlock(ctx, blk, [48]byte{}, [32]byte{})
	require.NoError(t, err)
	assert.Equal(t, true, slashable, "Expected attestation to be slashable")
	s = &Service{slasherClient: mockSlasher{slashAttestation: false}}
	slashable, err = s.IsSlashableBlock(ctx, blk, [48]byte{}, [32]byte{})
	require.NoError(t, err)
	assert.Equal(t, false, slashable, "Expected attestation to not be slashable")
}
