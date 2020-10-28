package testing

import (
	"testing"
)

func TestSZZStatic_Mainnet(t *testing.T) {
	t.Skip("// TODO see #7536 - unskip")
	runSSZStaticTests(t, "mainnet")
}
