package spectest

import (
	"testing"
)

func TestAttestationMinimal(t *testing.T) {
	t.Skip("Skip until 3960 merges")
	runAttestationTest(t, "minimal")
}
