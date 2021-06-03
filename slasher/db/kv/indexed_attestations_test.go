package kv

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

type testStruct struct {
	idxAtt *ethpb.IndexedAttestation
}

var tests []testStruct

func init() {
	tests = []testStruct{
		{
			idxAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{0},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
					Target: &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
				},
				Signature: []byte{1, 2},
			},
		},
		{
			idxAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{1, 2},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
					Target: &ethpb.Checkpoint{Epoch: 2, Root: make([]byte, 32)},
				},
				Signature: []byte{3, 4},
			},
		},
		{
			idxAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{0},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
					Target: &ethpb.Checkpoint{Epoch: 2, Root: make([]byte, 32)},
				},
				Signature: []byte{5, 6},
			},
		},
		{
			idxAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{0},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
					Target: &ethpb.Checkpoint{Epoch: 3, Root: make([]byte, 32)},
				},
				Signature: []byte{5, 6},
			},
		},
	}
}

func TestHasIndexedAttestation_NilDB(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	hasIdxAtt, err := db.HasIndexedAttestation(ctx, tests[0].idxAtt)
	require.NoError(t, err)
	require.Equal(t, false, hasIdxAtt)
}

func TestSaveIndexedAttestation(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	for _, tt := range tests {
		require.NoError(t, db.SaveIndexedAttestation(ctx, tt.idxAtt), "Save indexed attestation failed")

		exists, err := db.HasIndexedAttestation(ctx, tt.idxAtt)
		require.NoError(t, err, "Failed to get indexed attestation")
		require.Equal(t, true, exists, "Expected to find saved attestation in DB")
	}
}

func TestIndexedAttestationsWithPrefix(t *testing.T) {
	type prefixTestStruct struct {
		name           string
		attsInDB       []*ethpb.IndexedAttestation
		targetEpoch    types.Epoch
		searchPrefix   []byte
		expectedResult []*ethpb.IndexedAttestation
	}
	prefixTests := []prefixTestStruct{
		{
			name: "single item, same sig, should find one",
			attsInDB: []*ethpb.IndexedAttestation{
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target: &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
					},
					Signature: []byte{1, 2},
				},
			},
			searchPrefix: []byte{1, 2},
			targetEpoch:  1,
			expectedResult: []*ethpb.IndexedAttestation{
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target: &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
					},
					Signature: []byte{1, 2},
				},
			},
		},
		{
			name: "multiple item, same sig, should find 3",
			attsInDB: []*ethpb.IndexedAttestation{
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
						BeaconBlockRoot: []byte("hi there"),
					},
					Signature: []byte{1, 2, 3},
				},
				{
					AttestingIndices: []uint64{1},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
						BeaconBlockRoot: []byte("hi there 2"),
					},
					Signature: []byte{1, 2, 4},
				},
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
						BeaconBlockRoot: []byte("hi there 3"),
					},
					Signature: []byte{1, 2, 5},
				},
			},
			searchPrefix: []byte{1, 2},
			targetEpoch:  1,
			expectedResult: []*ethpb.IndexedAttestation{
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
						BeaconBlockRoot: []byte("hi there"),
					},
					Signature: []byte{1, 2, 3},
				},
				{
					AttestingIndices: []uint64{1},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
						BeaconBlockRoot: []byte("hi there 2"),
					},
					Signature: []byte{1, 2, 4},
				},
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
						BeaconBlockRoot: []byte("hi there 3"),
					},
					Signature: []byte{1, 2, 5},
				},
			},
		},
		{
			name: "multiple items, different sig and epoch, should find 2",
			attsInDB: []*ethpb.IndexedAttestation{
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
						BeaconBlockRoot: []byte("hi there"),
					},
					Signature: []byte{1, 2, 3},
				},
				{
					AttestingIndices: []uint64{1},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 2, Root: make([]byte, 32)},
						BeaconBlockRoot: []byte("hi there"),
					},
					Signature: []byte{1, 2, 4},
				},
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 3, Root: make([]byte, 32)},
						BeaconBlockRoot: []byte("hi there 3"),
					},
					Signature: []byte{1, 2, 5},
				},
				{
					AttestingIndices: []uint64{1},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 3, Root: make([]byte, 32)},
						BeaconBlockRoot: []byte("hi there 2"),
					},
					Signature: []byte{1, 3, 1},
				},
				{
					AttestingIndices: []uint64{1},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 2, Root: make([]byte, 32)},
						BeaconBlockRoot: []byte("hi there 2"),
					},
					Signature: []byte{0, 2, 4},
				},
				{
					AttestingIndices: []uint64{4},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 2, Root: make([]byte, 32)},
						BeaconBlockRoot: []byte("hi there 2"),
					},
					Signature: []byte{1, 2, 9},
				},
			},
			searchPrefix: []byte{1, 2},
			targetEpoch:  2,
			expectedResult: []*ethpb.IndexedAttestation{
				{
					AttestingIndices: []uint64{1},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 2, Root: make([]byte, 32)},
						BeaconBlockRoot: []byte("hi there"),
					},
					Signature: []byte{1, 2, 4},
				},
				{
					AttestingIndices: []uint64{4},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 2, Root: make([]byte, 32)},
						BeaconBlockRoot: []byte("hi there 2"),
					},
					Signature: []byte{1, 2, 9},
				},
			},
		},
		{
			name: "multiple items, different sigs, should not find any atts",
			attsInDB: []*ethpb.IndexedAttestation{
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 2, Root: make([]byte, 32)},
						BeaconBlockRoot: []byte("hi there"),
					},
					Signature: []byte{3, 5, 3},
				},
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 2, Root: make([]byte, 32)},
						BeaconBlockRoot: []byte("hi there"),
					},
					Signature: []byte{3, 5, 3},
				},
				{
					AttestingIndices: []uint64{1},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
						BeaconBlockRoot: []byte("hi there 2"),
					},
					Signature: []byte{1, 2, 4},
				},
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
						BeaconBlockRoot: []byte("hi there 3"),
					},
					Signature: []byte{1, 2, 5},
				},
			},
			searchPrefix: []byte{3, 5},
			targetEpoch:  1,
		},
	}
	for _, tt := range prefixTests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupDB(t)
			ctx := context.Background()

			require.NoError(t, db.SaveIndexedAttestations(ctx, tt.attsInDB), "Save indexed attestation failed")
			for _, att := range tt.attsInDB {
				found, err := db.HasIndexedAttestation(ctx, att)
				require.NoError(t, err)
				require.Equal(t, true, found, "Expected to save %v", att)
			}

			idxAtts, err := db.IndexedAttestationsWithPrefix(ctx, tt.targetEpoch, tt.searchPrefix)
			require.NoError(t, err, "Failed to get indexed attestation")
			require.DeepSSZEqual(t, tt.expectedResult, idxAtts)
		})
	}
}

func TestIndexedAttestationsForTarget(t *testing.T) {
	type prefixTestStruct struct {
		name           string
		attsInDB       []*ethpb.IndexedAttestation
		targetEpoch    types.Epoch
		expectedResult []*ethpb.IndexedAttestation
	}
	prefixTests := []prefixTestStruct{
		{
			name: "single item, should find one",
			attsInDB: []*ethpb.IndexedAttestation{
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target: &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
					},
					Signature: []byte{1, 2},
				},
			},
			targetEpoch: 1,
			expectedResult: []*ethpb.IndexedAttestation{
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target: &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
					},
					Signature: []byte{1, 2},
				},
			},
		},
		{
			name: "multiple items, same epoch, should find 3",
			attsInDB: []*ethpb.IndexedAttestation{
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 3, Root: make([]byte, 32)},
						BeaconBlockRoot: []byte("hi there"),
					},
					Signature: []byte{1, 2, 3},
				},
				{
					AttestingIndices: []uint64{1},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 3, Root: make([]byte, 32)},
						BeaconBlockRoot: []byte("hi there 2"),
					},
					Signature: []byte{1, 5, 4},
				},
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 3, Root: make([]byte, 32)},
						BeaconBlockRoot: []byte("hi there 3"),
					},
					Signature: []byte{8, 2, 5},
				},
			},
			targetEpoch: 3,
			expectedResult: []*ethpb.IndexedAttestation{
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 3, Root: make([]byte, 32)},
						BeaconBlockRoot: []byte("hi there"),
					},
					Signature: []byte{1, 2, 3},
				},
				{
					AttestingIndices: []uint64{1},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 3, Root: make([]byte, 32)},
						BeaconBlockRoot: []byte("hi there 2"),
					},
					Signature: []byte{1, 5, 4},
				},
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 3, Root: make([]byte, 32)},
						BeaconBlockRoot: []byte("hi there 3"),
					},
					Signature: []byte{8, 2, 5},
				},
			},
		},
		{
			name: "multiple items, different epochs, should not find any atts",
			attsInDB: []*ethpb.IndexedAttestation{
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
						BeaconBlockRoot: []byte("hi there"),
					},
					Signature: []byte{3, 5, 3},
				},
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 2, Root: make([]byte, 32)},
						BeaconBlockRoot: []byte("hi there"),
					},
					Signature: []byte{3, 5, 3},
				},
				{
					AttestingIndices: []uint64{1},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 3, Root: make([]byte, 32)},
						BeaconBlockRoot: []byte("hi there 2"),
					},
					Signature: []byte{1, 2, 4},
				},
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 5, Root: make([]byte, 32)},
						BeaconBlockRoot: []byte("hi there 3"),
					},
					Signature: []byte{1, 2, 5},
				},
			},
			targetEpoch: 4,
		},
	}
	for _, tt := range prefixTests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupDB(t)
			ctx := context.Background()

			require.NoError(t, db.SaveIndexedAttestations(ctx, tt.attsInDB), "Save indexed attestation failed")
			for _, att := range tt.attsInDB {
				found, err := db.HasIndexedAttestation(ctx, att)
				require.NoError(t, err)
				require.Equal(t, true, found, "Expected to save %v", att)
			}

			idxAtts, err := db.IndexedAttestationsForTarget(ctx, tt.targetEpoch)
			require.NoError(t, err, "Failed to get indexed attestation: %v", err)
			require.DeepSSZEqual(t, tt.expectedResult, idxAtts)
		})
	}
}

func TestDeleteIndexedAttestation(t *testing.T) {
	type deleteTestStruct struct {
		name       string
		attsInDB   []*ethpb.IndexedAttestation
		deleteAtts []*ethpb.IndexedAttestation
		foundArray []bool
	}
	deleteTests := []deleteTestStruct{
		{
			name: "single item, should delete all",
			attsInDB: []*ethpb.IndexedAttestation{
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target: &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
					},
					Signature: []byte{1, 2},
				},
			},
			deleteAtts: []*ethpb.IndexedAttestation{
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target: &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
					},
					Signature: []byte{1, 2},
				},
			},
			foundArray: []bool{false},
		},
		{
			name: "multiple items, should delete 2",
			attsInDB: []*ethpb.IndexedAttestation{
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target: &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
					},
					Signature: []byte{1, 2},
				},
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target: &ethpb.Checkpoint{Epoch: 3, Root: make([]byte, 32)},
					},
					Signature: []byte{2, 4},
				},
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target: &ethpb.Checkpoint{Epoch: 4, Root: make([]byte, 32)},
					},
					Signature: []byte{3, 5},
				},
			},
			deleteAtts: []*ethpb.IndexedAttestation{
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target: &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
					},
					Signature: []byte{1, 2},
				},
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target: &ethpb.Checkpoint{Epoch: 4, Root: make([]byte, 32)},
					},
					Signature: []byte{3, 5},
				},
			},
			foundArray: []bool{false, true, false},
		},
		{
			name: "multiple similar items, should delete 1",
			attsInDB: []*ethpb.IndexedAttestation{
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target: &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
					},
					Signature: []byte{1, 2, 2},
				},
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target: &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
					},
					Signature: []byte{1, 2, 3},
				},
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target: &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
					},
					Signature: []byte{1, 2, 4},
				},
			},
			deleteAtts: []*ethpb.IndexedAttestation{
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target: &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
					},
					Signature: []byte{1, 2, 3},
				},
			},
			foundArray: []bool{true, false, true},
		},
		{
			name: "should not delete any if not in list",
			attsInDB: []*ethpb.IndexedAttestation{
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target: &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
					},
					Signature: []byte{1, 2, 2},
				},
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target: &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
					},
					Signature: []byte{1, 2, 3},
				},
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target: &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
					},
					Signature: []byte{1, 2, 4},
				},
			},
			deleteAtts: []*ethpb.IndexedAttestation{
				{
					AttestingIndices: []uint64{3},
					Data: &ethpb.AttestationData{
						Source: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target: &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
					},
					Signature: []byte{1, 2, 6},
				},
			},
			foundArray: []bool{true, true, true},
		},
	}
	for _, tt := range deleteTests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupDB(t)
			ctx := context.Background()

			require.NoError(t, db.SaveIndexedAttestations(ctx, tt.attsInDB), "Save indexed attestation failed")

			for _, att := range tt.attsInDB {
				found, err := db.HasIndexedAttestation(ctx, att)
				require.NoError(t, err)
				require.Equal(t, true, found, "Expected to save %v", att)
			}

			for _, att := range tt.deleteAtts {
				require.NoError(t, db.DeleteIndexedAttestation(ctx, att))
			}

			for i, att := range tt.attsInDB {
				found, err := db.HasIndexedAttestation(ctx, att)
				require.NoError(t, err)
				require.Equal(t, tt.foundArray[i], found)
			}
		})
	}
}

func TestHasIndexedAttestation(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	for _, tt := range tests {
		exists, err := db.HasIndexedAttestation(ctx, tt.idxAtt)
		require.NoError(t, err)
		require.Equal(t, false, exists, "has indexed attestation should return false for indexed attestations that are not in db")

		require.NoError(t, db.SaveIndexedAttestation(ctx, tt.idxAtt), "Save indexed attestation failed")
	}

	for _, tt := range tests {
		exists, err := db.HasIndexedAttestation(ctx, tt.idxAtt)
		require.NoError(t, err)
		require.Equal(t, true, exists)
	}
}

func TestPruneHistoryIndexedAttestation(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	for _, tt := range tests {
		require.NoError(t, db.SaveIndexedAttestation(ctx, tt.idxAtt), "Save indexed attestation failed")

		found, err := db.HasIndexedAttestation(ctx, tt.idxAtt)
		require.NoError(t, err, "Failed to get indexed attestation")
		require.Equal(t, true, found, "Expected to find attestation in DB")
	}
	currentEpoch := types.Epoch(2)
	historyToKeep := types.Epoch(1)
	require.NoError(t, db.PruneAttHistory(ctx, currentEpoch, historyToKeep), "Failed to prune")

	for _, tt := range tests {
		exists, err := db.HasIndexedAttestation(ctx, tt.idxAtt)
		require.NoError(t, err)

		if tt.idxAtt.Data.Target.Epoch > currentEpoch-historyToKeep {
			require.Equal(t, true, exists, "Expected to find attestation newer than prune age in DB")
		} else {
			require.Equal(t, false, exists, "Expected to not find attestation older than prune age in DB")
		}
	}
}
