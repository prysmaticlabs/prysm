package testing

import (
	"testing"
)

func TestSZZStatic_Mainnet(t *testing.T) {
	t.Skip("Skipped due to upstream configuration bug. See: https://github.com/prysmaticlabs/prysm/issues/7277")
	runSSZStaticTests(t, "mainnet")
}
