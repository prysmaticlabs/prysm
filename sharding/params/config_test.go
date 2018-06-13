package params

import (
	"math/big"
	"testing"
)

func TestNotaryDeposit(t *testing.T) {
	want, err := new(big.Int).SetString("1000000000000000000000", 10) // 1000 ETH
	if !err {
		t.Fatalf("Failed to setup test")
	}
	if DefaultConfig.NotaryDeposit.Cmp(want) != 0 {
		t.Errorf("Notary deposit size incorrect. Wanted %d, got %d", want, DefaultConfig.NotaryDeposit)
	}
}

func TestPeriodLength(t *testing.T) {
	if DefaultConfig.PeriodLength != 5 {
		t.Errorf("Shard count incorrect. Wanted %d, got %d", 5, DefaultConfig.PeriodLength)
	}
}

func TestNotaryLockupLength(t *testing.T) {
	if DefaultConfig.NotaryLockupLength != 16128 {
		t.Errorf("Shard count incorrect. Wanted %d, got %d", 16128, DefaultConfig.NotaryLockupLength)
	}
}

func TestProposerLockupLength(t *testing.T) {
	if DefaultConfig.ProposerLockupLength != 48 {
		t.Errorf("Shard count incorrect. Wanted %d, got %d", 48, DefaultConfig.ProposerLockupLength)
	}
}

func TestNotaryCommitteeSize(t *testing.T) {
	if DefaultConfig.NotaryCommitteeSize != 135 {
		t.Errorf("Shard count incorrect. Wanted %d, got %d", 135, DefaultConfig.NotaryCommitteeSize)
	}
}

func TestNotaryQuorumSize(t *testing.T) {
	if DefaultConfig.NotaryQuorumSize != 90 {
		t.Errorf("Shard count incorrect. Wanted %d, got %d", 90, DefaultConfig.NotaryQuorumSize)
	}
}

func TestNotaryChallengePeriod(t *testing.T) {
	if DefaultConfig.NotaryChallengePeriod != 25 {
		t.Errorf("Shard count incorrect. Wanted %d, got %d", 25, DefaultConfig.NotaryChallengePeriod)
	}
}
