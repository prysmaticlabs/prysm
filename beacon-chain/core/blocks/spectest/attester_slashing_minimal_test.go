package spectest

import (
	"testing"
)

func TestAttesterSlashingMinimal(t *testing.T) {
	t.Skip("Skip until 3960 merges")
	runAttesterSlashingTest(t, "minimal")
}
