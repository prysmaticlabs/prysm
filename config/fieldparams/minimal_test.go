// +build minimal

package field_params_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/config/params"
)

func TestFieldParametersValues(t *testing.T) {
	params.UseMinimalConfig()
	testFieldParametersMatchConfig(t)
}
