package spectest

import (
	"testing"
)

func TestSlashingsMinimal(t *testing.T) {
	t.Skip("Disabled until v0.9.0 (#3865) completes")
	runSlashingsTests(t, "minimal")
}
