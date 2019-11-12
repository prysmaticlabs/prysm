package testing

import (
	"testing"
)

func TestSSZStatic_Minimal(t *testing.T) {
	t.Skip("Skip until 3960 merges")
	runSSZStaticTests(t, "minimal")
}
