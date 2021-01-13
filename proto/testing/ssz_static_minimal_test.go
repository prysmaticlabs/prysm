package testing

import (
	"testing"
)

func TestSSZStatic_Minimal(t *testing.T) {
	t.Skip("We'll need to generate spec test for new hardfork configs")
	runSSZStaticTests(t, "minimal")
}
