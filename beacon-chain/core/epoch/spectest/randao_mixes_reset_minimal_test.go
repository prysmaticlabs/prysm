package spectest

import "testing"

func TestRandaoMixesResetMinimal(t *testing.T) {
	runRandaoMixesResetTests(t, "minimal")
}
