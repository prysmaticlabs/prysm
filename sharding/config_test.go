package sharding

import (
	"math/big"
	"testing"
)

func TestDepositSize(t *testing.T) {
	want, err := new(big.Int).SetString("100000000000000000000", 10) // 100 ETH
	if !err {
		t.Fatalf("Failed to setup test")
	}
	if DepositSize.Cmp(want) != 0 {
		t.Errorf("depositSize incorrect. Wanted %d, got %d", want, DepositSize)
	}
}
