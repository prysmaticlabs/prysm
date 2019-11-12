package testing

import (
	"testing"
)

func TestSZZStatic_Mainnet(t *testing.T) {
	t.Skip("Skip until 3960 merges")
	runSSZStaticTests(t, "mainnet")
}
