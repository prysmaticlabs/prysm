//go:build !minimal
// +build !minimal

package field_params_test

import (
	"github.com/prysmaticlabs/prysm/testing/require"
	"testing"

	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
)

func TestFieldParametersValues(t *testing.T) {
	min, err := params.Registry.GetByName(params.MainnetName)
	require.NoError(t, err)
	undo, err := params.Registry.SetActiveWithUndo(min)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, undo())
	}()
	require.Equal(t, "mainnet", fieldparams.Preset)
	testFieldParametersMatchConfig(t)
}
