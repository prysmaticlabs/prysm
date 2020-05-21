package testing

import (
	"testing"
)

func TestSZZStatic_Mainnet(t *testing.T) {
	t.Skip("Skipping until 5935 is resolved, this requires pointing spec test to the latest version")

	runSSZStaticTests(t, "mainnet")
}
