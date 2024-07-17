package lightclient

import (
	"testing"

	ethpbv1 "github.com/prysmaticlabs/prysm/v5/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v5/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
)

func TestIsBetterUpdate(t *testing.T) {
	testCases := []struct {
		name           string
		oldUpdate      *ethpbv2.LightClientUpdate
		newUpdate      *ethpbv2.LightClientUpdate
		expectedResult bool
	}{
		{
			name: "new has supermajority but old doesn't",
			oldUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b01111100, 0b1},
				},
			},
			newUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b11111100, 0b1},
				},
			},
			expectedResult: true,
		},
		{
			name: "old has supermajority but new doesn't",
			oldUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b11111100, 0b1},
				},
			},
			newUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b01111100, 0b1},
				},
			},
			expectedResult: false,
		},
		{
			name:           "new doesn't have supermajority and newNumActiveParticipants is greater than oldNumActiveParticipants",
			oldUpdate:      &ethpbv2.LightClientUpdate{},
			newUpdate:      &ethpbv2.LightClientUpdate{},
			expectedResult: true,
		},
		{
			name:           "new doesn't have supermajority and newNumActiveParticipants is lesser than oldNumActiveParticipants",
			oldUpdate:      &ethpbv2.LightClientUpdate{},
			newUpdate:      &ethpbv2.LightClientUpdate{},
			expectedResult: false,
		},
		{
			name:           "new has relevant sync committee but old doesn't",
			oldUpdate:      &ethpbv2.LightClientUpdate{},
			newUpdate:      &ethpbv2.LightClientUpdate{},
			expectedResult: true,
		},
		{
			name:           "old has relevant sync committee but new doesn't",
			oldUpdate:      &ethpbv2.LightClientUpdate{},
			newUpdate:      &ethpbv2.LightClientUpdate{},
			expectedResult: false,
		},
		{
			name:           "new has finality but old doesn't",
			oldUpdate:      &ethpbv2.LightClientUpdate{},
			newUpdate:      &ethpbv2.LightClientUpdate{},
			expectedResult: true,
		},
		{
			name:           "old has finality but new doesn't",
			oldUpdate:      &ethpbv2.LightClientUpdate{},
			newUpdate:      &ethpbv2.LightClientUpdate{},
			expectedResult: false,
		},
		{
			name:           "new has finality and sync committee finality both but old doesn't have sync committee finality",
			oldUpdate:      &ethpbv2.LightClientUpdate{},
			newUpdate:      &ethpbv2.LightClientUpdate{},
			expectedResult: true,
		},
		{
			name:           "new has finality but doesn't have sync committee finality and old has sync committee finality",
			oldUpdate:      &ethpbv2.LightClientUpdate{},
			newUpdate:      &ethpbv2.LightClientUpdate{},
			expectedResult: false,
		},
		{
			name:           "new has more active participants than old",
			oldUpdate:      &ethpbv2.LightClientUpdate{},
			newUpdate:      &ethpbv2.LightClientUpdate{},
			expectedResult: true,
		},
		{
			name:           "new has less active participants than old",
			oldUpdate:      &ethpbv2.LightClientUpdate{},
			newUpdate:      &ethpbv2.LightClientUpdate{},
			expectedResult: false,
		},
		{
			name:           "new's attested header's slot is lesser than old's attested header's slot",
			oldUpdate:      &ethpbv2.LightClientUpdate{},
			newUpdate:      &ethpbv2.LightClientUpdate{},
			expectedResult: true,
		},
		{
			name:           "new's attested header's slot is greater than old's attested header's slot",
			oldUpdate:      &ethpbv2.LightClientUpdate{},
			newUpdate:      &ethpbv2.LightClientUpdate{},
			expectedResult: false,
		},
		{
			name:           "none of the above conditions are met and new's signature slot is lesser than old's signature slot",
			oldUpdate:      &ethpbv2.LightClientUpdate{},
			newUpdate:      &ethpbv2.LightClientUpdate{},
			expectedResult: true,
		},
		{
			name:           "none of the above conditions are met and new's signature slot is greater than old's signature slot",
			oldUpdate:      &ethpbv2.LightClientUpdate{},
			newUpdate:      &ethpbv2.LightClientUpdate{},
			expectedResult: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expectedResult, IsBetterUpdate(testCase.newUpdate, testCase.oldUpdate))
		})
	}
}
