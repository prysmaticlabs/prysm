package migration

import (
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

var (
	slot           = types.Slot(1)
	validatorIndex = types.ValidatorIndex(1)
	parentRoot     = bytesutil.PadTo([]byte("parentroot"), 32)
	stateRoot      = bytesutil.PadTo([]byte("stateroot"), 32)
	signature      = bytesutil.PadTo([]byte("signature"), 96)
	bodyRoot       = bytesutil.PadTo([]byte("bodyroot"), 32)
)

func Test_V1ProposerSlashingToV1Alpha1(t *testing.T) {
	v1Header := &ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			Slot:          slot,
			ProposerIndex: validatorIndex,
			ParentRoot:    parentRoot,
			StateRoot:     stateRoot,
			BodyRoot:      bodyRoot,
		},
		Signature: signature,
	}
	v1Slashing := &ethpb.ProposerSlashing{
		Header_1: v1Header,
		Header_2: v1Header,
	}

	alphaSlashing := V1ProposerSlashingToV1Alpha1(v1Slashing)
	alphaRoot, err := alphaSlashing.HashTreeRoot()
	require.NoError(t, err)
	v1Root, err := v1Slashing.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, alphaRoot, v1Root)
}
