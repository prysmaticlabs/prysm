package testing

import (
	"testing"
)

func TestSZZStatic_Mainnet(t *testing.T) {
	t.Skip()
	runSSZStaticTests(t, "mainnet")
}
