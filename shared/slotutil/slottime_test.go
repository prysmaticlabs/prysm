package slotutil

import (
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
)

func TestSlotsSinceGenesis(t *testing.T) {
	type args struct {
		genesis time.Time
	}
	tests := []struct {
		name string
		args args
		want uint64
	}{
		{
			name: "pre-genesis",
			args: args{
				genesis: roughtime.Now().Add(1 * time.Hour), // 1 hour in the future
			},
			want: 0,
		},
		{
			name: "post-genesis",
			args: args{
				genesis: roughtime.Now().Add(-5 * time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Second),
			},
			want: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SlotsSinceGenesis(tt.args.genesis); got != tt.want {
				t.Errorf("SlotsSinceGenesis() = %v, want %v", got, tt.want)
			}
		})
	}
}
