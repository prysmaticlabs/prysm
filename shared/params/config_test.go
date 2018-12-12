package params

import (
	"testing"
)

func TestSpecialRecordTypes(t *testing.T) {
	tests := []struct {
		a SpecialRecordType
		b int
	}{
		{a: Logout, b: 0},
		{a: CasperSlashing, b: 1},
		{a: ProposerSlashing, b: 2},
		{a: DepositProof, b: 3},
	}
	for _, tt := range tests {
		if int(tt.a) != tt.b {
			t.Errorf("Incorrect special record types. Wanted: %d, Got: %d", int(tt.a), tt.b)
		}
	}
}

func TestValidatorSetDeltaFlags(t *testing.T) {
	tests := []struct {
		a ValidatorSetDeltaFlags
		b int
	}{
		{a: Entry, b: 0},
		{a: Exit, b: 1},
	}
	for _, tt := range tests {
		if int(tt.a) != tt.b {
			t.Errorf("Incorrect validator set delta flags. Wanted: %d, Got: %d", int(tt.a), tt.b)
		}
	}
}

func TestOverrideBeaconConfig(t *testing.T) {
	cfg := BeaconConfig()
	cfg.ShardCount = 5
	OverrideBeaconConfig(cfg)
	if c := BeaconConfig(); c.ShardCount != 5 {
		t.Errorf("Shardcount in BeaconConfig incorrect. Wanted %d, got %d", 5, c.ShardCount)
	}
}
