package testing

import (
	"testing"
)

func TestSSZStatic_Minimal(t *testing.T) {
	t.Skip("Disabled until v0.9.0 (#3865) completes")

	runSSZStaticTests(t, "minimal")
}
