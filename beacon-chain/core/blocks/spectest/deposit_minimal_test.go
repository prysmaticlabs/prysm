package spectest

import (
	"testing"
)

func TestDepositMinimalYaml(t *testing.T) {
	runDepositTest(t, "minimal")
}
