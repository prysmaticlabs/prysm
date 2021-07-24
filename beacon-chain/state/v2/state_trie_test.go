package v2_test

import (
	"testing"

	stateAltair "github.com/prysmaticlabs/prysm/beacon-chain/state/v2"
	statepb "github.com/prysmaticlabs/prysm/proto/prysm/v2/state"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestInitializeFromProto(t *testing.T) {
	type test struct {
		name  string
		state *statepb.BeaconStateAltair
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
			state: &statepb.BeaconStateAltair{
				Slot:       4,
				Validators: nil,
			},
		},
		{
			name:  "empty state",
			state: &statepb.BeaconStateAltair{},
		},
		// TODO: Add full state. Blocked by testutil migration.
	}
	for _, tt := range initTests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := stateAltair.InitializeFromProto(tt.state)
			if tt.error != "" {
				require.ErrorContains(t, tt.error, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestInitializeFromProtoUnsafe(t *testing.T) {
	type test struct {
		name  string
		state *statepb.BeaconStateAltair
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
			state: &statepb.BeaconStateAltair{
				Slot:       4,
				Validators: nil,
			},
		},
		{
			name:  "empty state",
			state: &statepb.BeaconStateAltair{},
		},
		// TODO: Add full state. Blocked by testutil migration.
	}
	for _, tt := range initTests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := stateAltair.InitializeFromProtoUnsafe(tt.state)
			if tt.error != "" {
				assert.ErrorContains(t, tt.error, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
