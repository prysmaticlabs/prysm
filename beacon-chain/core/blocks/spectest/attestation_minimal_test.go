package spectest

import (
	"testing"
)

func TestAttestationMinimal(t *testing.T) {
	t.Skip("Skipping until last stage of 5119")
	runAttestationTest(t, "minimal")
}
