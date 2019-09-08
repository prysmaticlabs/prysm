package spectest

import (
	"testing"
)

func TestAttesterSlashingMinimal(t *testing.T) {
	runAttesterSlashingTest(t, "minimal")
}
