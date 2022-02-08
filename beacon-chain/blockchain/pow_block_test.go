package blockchain

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/prysmaticlabs/prysm/config/params"
)

func Test_validTerminalPowBlock(t *testing.T) {
	tests := []struct {
		name              string
		currentDifficulty *uint256.Int
		parentDifficulty  *uint256.Int
		ttd               uint64
		want              bool
	}{
		{
			name:              "current > ttd, parent > ttd",
			currentDifficulty: uint256.NewInt(2),
			parentDifficulty:  uint256.NewInt(2),
			ttd:               1,
			want:              false,
		},
		{
			name:              "current < ttd, parent < ttd",
			currentDifficulty: uint256.NewInt(2),
			parentDifficulty:  uint256.NewInt(2),
			ttd:               3,
			want:              false,
		},
		{
			name:              "current == ttd, parent == ttd",
			currentDifficulty: uint256.NewInt(2),
			parentDifficulty:  uint256.NewInt(2),
			ttd:               2,
			want:              false,
		},
		{
			name:              "current > ttd, parent == ttd",
			currentDifficulty: uint256.NewInt(2),
			parentDifficulty:  uint256.NewInt(1),
			ttd:               1,
			want:              false,
		},
		{
			name:              "current == ttd, parent < ttd",
			currentDifficulty: uint256.NewInt(2),
			parentDifficulty:  uint256.NewInt(1),
			ttd:               2,
			want:              true,
		},
		{
			name:              "current > ttd, parent < ttd",
			currentDifficulty: uint256.NewInt(3),
			parentDifficulty:  uint256.NewInt(1),
			ttd:               2,
			want:              true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := params.BeaconConfig()
			cfg.TerminalTotalDifficulty = uint256.NewInt(tt.ttd)
			params.OverrideBeaconConfig(cfg)
			if got := validTerminalPowBlock(tt.currentDifficulty, tt.parentDifficulty); got != tt.want {
				t.Errorf("validTerminalPowBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}
