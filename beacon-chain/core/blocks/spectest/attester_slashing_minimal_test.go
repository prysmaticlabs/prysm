package spectest

import (
	"testing"
)

func TestAttesterSlashingMinimal(t *testing.T) {
	t.Skip("Skipping until last stage of 5119")
	runAttesterSlashingTest(t, "minimal")
}
