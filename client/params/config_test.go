package params

import (
	"math/big"
	"testing"
)

func TestAttesterDeposit(t *testing.T) {
	c := DefaultConfig()
	want, err := new(big.Int).SetString("1000000000000000000000", 10) // 1000 ETH
	if !err {
		t.Fatalf("Failed to setup test")
	}
	if c.AttesterDeposit.Cmp(want) != 0 {
		t.Errorf("Attester deposit size incorrect. Wanted %d, got %d", want, c.AttesterDeposit)
	}
}

func TestPeriodLength(t *testing.T) {
	c := DefaultConfig()
	if c.PeriodLength != 5 {
		t.Errorf("Shard count incorrect. Wanted %d, got %d", 5, c.PeriodLength)
	}
}

func TestAttesterLockupLength(t *testing.T) {
	c := DefaultConfig()
	if c.AttesterLockupLength != 16128 {
		t.Errorf("Shard count incorrect. Wanted %d, got %d", 16128, c.AttesterLockupLength)
	}
}

func TestProposerLockupLength(t *testing.T) {
	c := DefaultConfig()
	if c.ProposerLockupLength != 48 {
		t.Errorf("Shard count incorrect. Wanted %d, got %d", 48, c.ProposerLockupLength)
	}
}

func TestAttesterCommitteeSize(t *testing.T) {
	c := DefaultConfig()
	if c.AttesterCommitteeSize != 135 {
		t.Errorf("Shard count incorrect. Wanted %d, got %d", 135, c.AttesterCommitteeSize)
	}
}

func TestAttesterQuorumSize(t *testing.T) {
	c := DefaultConfig()
	if c.AttesterQuorumSize != 90 {
		t.Errorf("Shard count incorrect. Wanted %d, got %d", 90, c.AttesterQuorumSize)
	}
}

func TestAttesterChallengePeriod(t *testing.T) {
	c := DefaultConfig()
	if c.AttesterChallengePeriod != 25 {
		t.Errorf("Shard count incorrect. Wanted %d, got %d", 25, c.AttesterChallengePeriod)
	}
}
