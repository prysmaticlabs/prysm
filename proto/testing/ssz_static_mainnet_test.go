package testing

import (
	"testing"
)

func TestSZZStatic_Mainnet(t *testing.T) {
	t.Skip("We'll need to generate spec test for new hardfork configs")
	runSSZStaticTests(t, "mainnet")
}
