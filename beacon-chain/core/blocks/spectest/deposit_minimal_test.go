package spectest

import (
	"testing"
)

func TestDepositMinimalYaml(t *testing.T) {
	t.Skip("Disabled until v0.9.0 (#3865) completes")
	runDepositTest(t, "minimal")
}
