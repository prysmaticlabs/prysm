package slashings

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/config/params"
)

func TestSigningRootsDiffer(t *testing.T) {
	type args struct {
		existingSigningRoot [32]byte
		incomingSigningRoot [32]byte
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Empty existing signing root is slashable",
			args: args{
				existingSigningRoot: params.BeaconConfig().ZeroHash,
				incomingSigningRoot: [32]byte{1},
			},
			want: true,
		},
		{
			name: "Non-empty, different existing signing root is slashable",
			args: args{
				existingSigningRoot: [32]byte{2},
				incomingSigningRoot: [32]byte{1},
			},
			want: true,
		},
		{
			name: "Non-empty, same existing signing root and incoming signing root is not slashable",
			args: args{
				existingSigningRoot: [32]byte{2},
				incomingSigningRoot: [32]byte{2},
			},
			want: false,
		},
		{
			name: "Both empty are considered slashable",
			args: args{
				existingSigningRoot: params.BeaconConfig().ZeroHash,
				incomingSigningRoot: params.BeaconConfig().ZeroHash,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SigningRootsDiffer(tt.args.existingSigningRoot, tt.args.incomingSigningRoot); got != tt.want {
				t.Errorf("SigningRootsDiffer() = %v, want %v", got, tt.want)
			}
		})
	}
}
