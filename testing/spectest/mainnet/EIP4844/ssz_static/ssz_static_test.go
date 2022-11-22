package ssz_static

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/spectest/shared/eip4844/ssz_static"
)

func TestMainnet_EIP4844_SSZStatic(t *testing.T) {
	ssz_static.RunSSZStaticTests(t, "mainnet")
}
