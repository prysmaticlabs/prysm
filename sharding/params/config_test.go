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
	if DefaultShardConfig.NotaryDeposit.Cmp(want) != 0 {
		t.Errorf("Notary deposit size incorrect. Wanted %d, got %d", want, DefaultShardConfig.NotaryDeposit)
	}
}

func TestPeriodLength(t *testing.T) {
	if DefaultShardConfig.PeriodLength != 5 {
		t.Errorf("Shard count incorrect. Wanted %d, got %d", 5, DefaultShardConfig.PeriodLength)
	}
}

func TestNotaryLockupLength(t *testing.T) {
	if DefaultShardConfig.NotaryLockupLength != 16128 {
		t.Errorf("Shard count incorrect. Wanted %d, got %d", 16128, DefaultShardConfig.NotaryLockupLength)
	}
}

func TestProposerLockupLength(t *testing.T) {
	if DefaultShardConfig.ProposerLockupLength != 48 {
		t.Errorf("Shard count incorrect. Wanted %d, got %d", 48, DefaultShardConfig.ProposerLockupLength)
	}
}

func TestNotaryCommitteeSize(t *testing.T) {
	if DefaultShardConfig.NotaryCommitteeSize != 135 {
		t.Errorf("Shard count incorrect. Wanted %d, got %d", 135, DefaultShardConfig.NotaryCommitteeSize)
	}
}

func TestNotaryQuorumSize(t *testing.T) {
	if DefaultShardConfig.NotaryQuorumSize != 90 {
		t.Errorf("Shard count incorrect. Wanted %d, got %d", 90, DefaultShardConfig.NotaryQuorumSize)
	}
}

func TestNotaryChallengePeriod(t *testing.T) {
	if DefaultShardConfig.NotaryChallengePeriod != 25 {
		t.Errorf("Shard count incorrect. Wanted %d, got %d", 25, DefaultShardConfig.NotaryChallengePeriod)
	}
}
