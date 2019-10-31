package spectest

import (
	"testing"
)

func TestVoluntaryExitMinimal(t *testing.T) {
	t.Skip("Disabled until v0.9.0 (#3865) completes")
	runVoluntaryExitTest(t, "minimal")
}
