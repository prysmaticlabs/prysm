package sharding

import (
	"math/big"
	"testing"
)

func TestNotaryDeposit(t *testing.T) {
	want, err := new(big.Int).SetString("1000000000000000000000", 10) // 1000 ETH
	if !err {
		t.Fatalf("Failed to setup test")
	}
	if NotaryDeposit.Cmp(want) != 0 {
		t.Errorf("Notary deposit size incorrect. Wanted %d, got %d", want, NotaryDeposit)
	}
}

func TestProposerDeposit(t *testing.T) {
	want, err := new(big.Int).SetString("1000000000000000000", 10) // 1 ETH
	if !err {
		t.Fatalf("Failed to setup test")
	}
	if ProposerDeposit.Cmp(want) != 0 {
		t.Errorf("Proposer deposit size incorrect. Wanted %d, got %d", want, ProposerDeposit)
	}
}

func TestMinProposerBalance(t *testing.T) {
	want, err := new(big.Int).SetString("100000000000000000", 10) // 0.1 ETH
	if !err {
		t.Fatalf("Failed to setup test")
	}
	if MinProposerBalance.Cmp(want) != 0 {
		t.Errorf("Min proposer balance incorrect. Wanted %d, got %d", want, MinProposerBalance)
	}
}

func TestNotarySubsidy(t *testing.T) {
	want, err := new(big.Int).SetString("1000000000000000", 10) // 0.001 ETH
	if !err {
		t.Fatalf("Failed to setup test")
	}
	if NotarySubsidy.Cmp(want) != 0 {
		t.Errorf("Notary subsidy size incorrect. Wanted %d, got %d", want, NotarySubsidy)
	}
}
