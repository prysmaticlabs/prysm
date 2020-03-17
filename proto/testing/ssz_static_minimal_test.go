package testing

import (
	"testing"
)

func TestSSZStatic_Minimal(t *testing.T) {
	t.Skip("Skipping until last stage of 5119")
	runSSZStaticTests(t, "minimal")
}
