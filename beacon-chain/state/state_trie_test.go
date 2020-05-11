package state

import (
	"testing"

	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestInitializeFromProto(t *testing.T) {
	type test struct {
		name  string
		state *pbp2p.BeaconState
		error string
	}
	initTests := []test{
		{
			name:  "nil state",
			state: nil,
			error: "received nil state",
		},
		{
			name: "nil validators",
			state: &pbp2p.BeaconState{
				Slot:       4,
				Validators: nil,
			},
			error: "",
		},
		{
			name:  "empty state",
			state: &pbp2p.BeaconState{},
			error: "",
		},
	}
	for _, tt := range initTests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := InitializeFromProto(tt.state)
			if err != nil && err.Error() != tt.error {
				t.Errorf("Unexpected error, expected %v, recevied %v", tt.error, err)
			}
			if err == nil && tt.error != "" {
				t.Errorf("Expected error, expected %v, recevied %v", tt.error, err)
			}
		})
	}
}

func TestInitializeFromProtoUnsafe(t *testing.T) {
	type test struct {
		name  string
		state *pbp2p.BeaconState
		error string
	}
	initTests := []test{
		{
			name:  "nil state",
			state: nil,
			error: "received nil state",
		},
		{
			name: "nil validators",
			state: &pbp2p.BeaconState{
				Slot:       4,
				Validators: nil,
			},
			error: "",
		},
		{
			name:  "empty state",
			state: &pbp2p.BeaconState{},
			error: "",
		},
	}
	for _, tt := range initTests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := InitializeFromProtoUnsafe(tt.state)
			if err != nil && err.Error() != tt.error {
				t.Errorf("Unexpected error, expected %v, recevied %v", tt.error, err)
			}
			if err == nil && tt.error != "" {
				t.Errorf("Expected error, expected %v, recevied %v", tt.error, err)
			}
		})
	}
}
