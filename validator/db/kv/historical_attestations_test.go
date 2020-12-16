package kv

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestNewAttestationHistoryArray(t *testing.T) {
	ba := NewAttestationHistoryArray(0)
	assert.Equal(t, latestEpochWrittenSize+historySize, len(ba))
	ba = NewAttestationHistoryArray(params.BeaconConfig().WeakSubjectivityPeriod - 1)
	assert.Equal(t, latestEpochWrittenSize+historySize*params.BeaconConfig().WeakSubjectivityPeriod, uint64(len(ba)))
	ba = NewAttestationHistoryArray(params.BeaconConfig().WeakSubjectivityPeriod)
	assert.Equal(t, latestEpochWrittenSize+historySize, len(ba))
	ba = NewAttestationHistoryArray(params.BeaconConfig().WeakSubjectivityPeriod + 1)
	assert.Equal(t, latestEpochWrittenSize+historySize+historySize, len(ba))

}

func TestSizeChecks(t *testing.T) {
	require.ErrorContains(t, "is smaller then minimal size", EncHistoryData{}.assertSize())
	require.NoError(t, EncHistoryData{0, 1, 2, 3, 4, 5, 6, 7}.assertSize())
	require.ErrorContains(t, "is not a multiple of entry size", EncHistoryData{0, 1, 2, 3, 4, 5, 6, 7, 8}.assertSize())
	require.NoError(t, NewAttestationHistoryArray(0).assertSize())
	require.NoError(t, NewAttestationHistoryArray(1).assertSize())
	require.NoError(t, NewAttestationHistoryArray(params.BeaconConfig().WeakSubjectivityPeriod).assertSize())
	require.NoError(t, NewAttestationHistoryArray(params.BeaconConfig().WeakSubjectivityPeriod-1).assertSize())
}

func TestGetLatestEpochWritten(t *testing.T) {
	ctx := context.Background()
	ha := NewAttestationHistoryArray(0)
	ha[0] = 28
	lew, err := ha.GetLatestEpochWritten(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(28), lew)
}

func TestSetLatestEpochWritten(t *testing.T) {
	ctx := context.Background()
	ha := NewAttestationHistoryArray(0)
	lew, err := ha.SetLatestEpochWritten(ctx, 2828282828)
	require.NoError(t, err)
	bytes := lew[:latestEpochWrittenSize]
	assert.Equal(t, uint64(2828282828), bytesutil.FromBytes8(bytes))
}

func TestGetTargetData(t *testing.T) {
	ctx := context.Background()
	ha := NewAttestationHistoryArray(0)
	td, err := ha.GetTargetData(ctx, 0)
	require.NoError(t, err)
	assert.DeepEqual(t, emptyHistoryData(), td)
	td, err = ha.GetTargetData(ctx, 1)
	require.NoError(t, err)
	var nilHist *HistoryData
	require.Equal(t, nilHist, td)
}

func TestSetTargetData_MarksUnattestedEpochsInBetween(t *testing.T) {
	ctx := context.Background()
	h1 := NewAttestationHistoryArray(0)

	// Write mark target 1, source 0 as attested.
	sr1 := [32]byte{}
	copy(sr1[:], "1")
	h2, err := h1.SetTargetData(ctx, 1, &HistoryData{
		Source:      0,
		SigningRoot: sr1[:],
	})
	require.NoError(t, err)

	// We mark target 50, source 49 as attested.
	sr2 := [32]byte{}
	copy(sr2[:], "50")
	highestEpoch := uint64(50)
	h3, err := h2.SetTargetData(ctx, highestEpoch, &HistoryData{
		Source:      highestEpoch - 1,
		SigningRoot: sr2[:],
	})
	require.NoError(t, err)

	// We verify we have a history for target 1 and for target 50.
	lowestData, err := h3.GetTargetData(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, lowestData)
	require.Equal(t, uint64(0), lowestData.Source)

	highestData, err := h3.GetTargetData(ctx, highestEpoch)
	require.NoError(t, err)
	require.NotNil(t, highestData)
	require.Equal(t, highestEpoch-1, highestData.Source)

	// We check all other epochs in between have an empty attesting history.
	for i := uint64(2); i < highestEpoch; i++ {
		data, err := h3.GetTargetData(ctx, i)
		require.NoError(t, err)
		require.Equal(t, true, data.IsEmpty())
	}
}

func TestSetTargetData(t *testing.T) {
	ctx := context.Background()
	type testStruct struct {
		name        string
		enc         EncHistoryData
		target      uint64
		source      uint64
		signingRoot []byte
		expected    EncHistoryData
		error       string
	}
	tests := []testStruct{
		{
			name:        "empty enc",
			enc:         EncHistoryData{},
			target:      0,
			source:      100,
			signingRoot: []byte{1, 2, 3},
			expected:    (EncHistoryData)(nil),
			error:       "encapsulated data size",
		},
		{
			name:        "new enc",
			enc:         NewAttestationHistoryArray(0),
			target:      0,
			source:      100,
			signingRoot: []byte{1, 2, 3},
			expected:    EncHistoryData{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x64, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x2, 0x3, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
			error:       "",
		},
		{
			name:        "higher target",
			enc:         NewAttestationHistoryArray(0),
			target:      2,
			source:      100,
			signingRoot: []byte{1, 2, 3},
			expected:    EncHistoryData{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x64, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x2, 0x3, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
			error:       "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc, err := tt.enc.SetTargetData(ctx,
				tt.target,
				&HistoryData{
					Source:      tt.source,
					SigningRoot: tt.signingRoot,
				})
			if tt.error == "" {
				require.NoError(t, err)
				td, err := enc.GetTargetData(ctx, tt.target)
				require.NoError(t, err)
				require.DeepEqual(t, bytesutil.PadTo(tt.signingRoot, 32), td.SigningRoot)
				require.Equal(t, tt.source, td.Source)
				return
			}
			assert.ErrorContains(t, tt.error, err)
			require.DeepEqual(t, tt.expected, enc)

		})
	}
}
