package slotutil

import (
	"testing"
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/timeutils"
)

func TestSlotsSinceGenesis(t *testing.T) {
	type args struct {
		genesis time.Time
	}
	tests := []struct {
		name string
		args args
		want types.Slot
	}{
		{
			name: "pre-genesis",
			args: args{
				genesis: timeutils.Now().Add(1 * time.Hour), // 1 hour in the future
			},
			want: 0,
		},
		{
			name: "post-genesis",
			args: args{
				genesis: timeutils.Now().Add(-5 * time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Second),
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

func TestAbsoluteValueSlotDifference(t *testing.T) {
	type args struct {
		x types.Slot
		y types.Slot
	}
	tests := []struct {
		name string
		args args
		want uint64
	}{
		{
			name: "x_<_y",
			args: args{
				x: types.Slot(3),
				y: types.Slot(4),
			},
			want: 1,
		},
		{
			name: "x_>_y",
			args: args{
				x: types.Slot(100),
				y: types.Slot(4),
			},
			want: 96,
		},
		{
			name: "x_==_y",
			args: args{
				x: types.Slot(100),
				y: types.Slot(100),
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AbsoluteValueSlotDifference(tt.args.x, tt.args.y); got != tt.want {
				t.Errorf("AbsoluteValueSlotDifference() = %v, want %v", got, tt.want)
			}
		})
	}
}
