// Package spectest contains all conformity specification tests
// for epoch processing according to the eth2 spec.
package spectest

import (
	"testing"
)

func TestRegistryUpdatesMainnet(t *testing.T) {
	runRegistryUpdatesTests(t, "mainnet")
}
