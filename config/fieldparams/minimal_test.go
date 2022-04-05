//go:build minimal
// +build minimal

package field_params_test

import (
	"testing"

	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/testing/assert"
)

func TestFieldParametersValues(t *testing.T) {
	params.UseMinimalConfig()
	assert.Equal(t, "minimal", fieldparams.Preset)
	testFieldParametersMatchConfig(t)
}
