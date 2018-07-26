package params

import (
	"math/big"
	"testing"
)

func TestAttesterDeposit(t *testing.T) {
	want, err := new(big.Int).SetString("1000000000000000000000", 10) // 1000 ETH
	if !err {
		t.Fatalf("Failed to setup test")
	}
	if DefaultConfig.AttesterDeposit.Cmp(want) != 0 {
		t.Errorf("Attester deposit size incorrect. Wanted %d, got %d", want, DefaultConfig.AttesterDeposit)
	}
}

func TestPeriodLength(t *testing.T) {
	if DefaultConfig.PeriodLength != 5 {
		t.Errorf("Shard count incorrect. Wanted %d, got %d", 5, DefaultConfig.PeriodLength)
	}
}

func TestAttesterLockupLength(t *testing.T) {
	if DefaultConfig.AttesterLockupLength != 16128 {
		t.Errorf("Shard count incorrect. Wanted %d, got %d", 16128, DefaultConfig.AttesterLockupLength)
	}
}

func TestProposerLockupLength(t *testing.T) {
	if DefaultConfig.ProposerLockupLength != 48 {
		t.Errorf("Shard count incorrect. Wanted %d, got %d", 48, DefaultConfig.ProposerLockupLength)
	}
}

func TestAttesterCommitteeSize(t *testing.T) {
	if DefaultConfig.AttesterCommitteeSize != 135 {
		t.Errorf("Shard count incorrect. Wanted %d, got %d", 135, DefaultConfig.AttesterCommitteeSize)
	}
}

func TestAttesterQuorumSize(t *testing.T) {
	if DefaultConfig.AttesterQuorumSize != 90 {
		t.Errorf("Shard count incorrect. Wanted %d, got %d", 90, DefaultConfig.AttesterQuorumSize)
	}
}

func TestAttesterChallengePeriod(t *testing.T) {
	if DefaultConfig.AttesterChallengePeriod != 25 {
		t.Errorf("Shard count incorrect. Wanted %d, got %d", 25, DefaultConfig.AttesterChallengePeriod)
	}
}
