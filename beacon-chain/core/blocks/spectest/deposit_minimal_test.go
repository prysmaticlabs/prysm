package spectest

import (
	"testing"
)

func TestDepositMinimalYaml(t *testing.T) {
	t.Skip("We'll need to generate spec test for new hardfork configs")
	runDepositTest(t, "minimal")
}
