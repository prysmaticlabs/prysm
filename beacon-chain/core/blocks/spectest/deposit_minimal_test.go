package spectest

import (
	"testing"
)

func TestDepositMinimal(t *testing.T) {
	runDepositTest(t, "minimal")
}
