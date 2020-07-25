package testing

import (
	"testing"
)

func TestSZZStatic_Mainnet(t *testing.T) {
	runSSZStaticTests(t, "mainnet")
}

func BenchmarkSSZStatic_Mainnet(b *testing.B) {
	runSSZStaticTests(b, "mainnet")
}