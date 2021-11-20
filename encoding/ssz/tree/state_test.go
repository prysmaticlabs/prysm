package tree

import (
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/io/file"
	v2 "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestStateTree(t *testing.T) {
	enc, err := file.ReadFileAsBytes("/tmp/state.ssz")
	require.NoError(t, err)
	stateAltair := &v2.BeaconStateAltair{}
	require.NoError(t, stateAltair.UnmarshalSSZ(enc))
	fmt.Println(stateAltair.Slot)
	_, err = stateAltair.GetTree()
	require.NoError(t, err)
}
