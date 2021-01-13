// Package spectest contains all conformity specification tests
// for epoch processing according to the eth2 spec.
package spectest

import (
	"testing"
)

func TestRegistryUpdatesMainnet(t *testing.T) {
	t.Skip("We'll need to generate spec test for new hardfork configs")
	runRegistryUpdatesTests(t, "mainnet")
}
