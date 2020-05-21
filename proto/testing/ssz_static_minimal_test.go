package testing

import (
	"testing"
)

func TestSSZStatic_Minimal(t *testing.T) {
	t.Skip("Skipping until 5935 is resolved, this requires pointing spec test to the latest version")

	runSSZStaticTests(t, "minimal")
}
