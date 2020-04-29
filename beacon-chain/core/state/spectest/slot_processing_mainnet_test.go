// Package spectest contains all conformity specification tests
// for slot processing logic according to the eth2 beacon spec.
package spectest

import (
	"testing"
)

func TestSlotProcessingMainnet(t *testing.T) {
	runSlotProcessingTests(t, "mainnet")
}
