package testing

import (
	"testing"
)

func TestSSZStatic_Minimal(t *testing.T) {
	t.Skip("This test suite requires --define ssz=minimal to be provided and there isn't a great way to do that without breaking //...")
	runSSZStaticTests(t, "minimal")
}
