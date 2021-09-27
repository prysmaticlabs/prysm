package kv

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestNewAttestationHistoryArray(t *testing.T) {
	ba := newDeprecatedAttestingHistory(0)
	assert.Equal(t, latestEpochWrittenSize+historySize, len(ba))
	ba = newDeprecatedAttestingHistory(params.BeaconConfig().WeakSubjectivityPeriod - 1)
	assert.Equal(t, latestEpochWrittenSize+historySize*params.BeaconConfig().WeakSubjectivityPeriod, types.Epoch(len(ba)))
	ba = newDeprecatedAttestingHistory(params.BeaconConfig().WeakSubjectivityPeriod)
	assert.Equal(t, latestEpochWrittenSize+historySize, len(ba))
	ba = newDeprecatedAttestingHistory(params.BeaconConfig().WeakSubjectivityPeriod + 1)
	assert.Equal(t, latestEpochWrittenSize+historySize+historySize, len(ba))

}

func TestSizeChecks(t *testing.T) {
	require.ErrorContains(t, "is smaller then minimal size", deprecatedEncodedAttestingHistory{}.assertSize())
	require.NoError(t, deprecatedEncodedAttestingHistory{0, 1, 2, 3, 4, 5, 6, 7}.assertSize())
	require.ErrorContains(t, "is not a multiple of entry size", deprecatedEncodedAttestingHistory{0, 1, 2, 3, 4, 5, 6, 7, 8}.assertSize())
	require.NoError(t, newDeprecatedAttestingHistory(0).assertSize())
	require.NoError(t, newDeprecatedAttestingHistory(1).assertSize())
	require.NoError(t, newDeprecatedAttestingHistory(params.BeaconConfig().WeakSubjectivityPeriod).assertSize())
	require.NoError(t, newDeprecatedAttestingHistory(params.BeaconConfig().WeakSubjectivityPeriod-1).assertSize())
}

func TestGetLatestEpochWritten(t *testing.T) {
	ctx := context.Background()
	ha := newDeprecatedAttestingHistory(0)
	ha[0] = 28
	lew, err := ha.getLatestEpochWritten(ctx)
	require.NoError(t, err)
	assert.Equal(t, types.Epoch(28), lew)
}

func TestSetLatestEpochWritten(t *testing.T) {
	ctx := context.Background()
	ha := newDeprecatedAttestingHistory(0)
	lew, err := ha.setLatestEpochWritten(ctx, 2828282828)
	require.NoError(t, err)
	bytes := lew[:latestEpochWrittenSize]
	assert.Equal(t, uint64(2828282828), bytesutil.FromBytes8(bytes))
}

func TestGetTargetData(t *testing.T) {
	ctx := context.Background()
	ha := newDeprecatedAttestingHistory(0)
	td, err := ha.getTargetData(ctx, 0)
	require.NoError(t, err)
	assert.DeepEqual(t, emptyHistoryData(), td)
	td, err = ha.getTargetData(ctx, 1)
	require.NoError(t, err)
	var nilHist *deprecatedHistoryData
	require.Equal(t, nilHist, td)
}

func TestSetTargetData(t *testing.T) {
	ctx := context.Background()
	type testStruct struct {
		name        string
		enc         deprecatedEncodedAttestingHistory
		target      types.Epoch
		source      types.Epoch
		signingRoot []byte
		expected    deprecatedEncodedAttestingHistory
		error       string
	}
	tests := []testStruct{
		{
			name:        "empty enc",
			enc:         deprecatedEncodedAttestingHistory{},
			target:      0,
			source:      100,
			signingRoot: []byte{1, 2, 3},
			expected:    deprecatedEncodedAttestingHistory(nil),
			error:       "encapsulated data size",
		},
		{
			name:        "new enc",
			enc:         newDeprecatedAttestingHistory(0),
			target:      0,
			source:      100,
			signingRoot: []byte{1, 2, 3},
			expected:    deprecatedEncodedAttestingHistory{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x64, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x2, 0x3, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
			error:       "",
		},
		{
			name:        "higher target",
			enc:         newDeprecatedAttestingHistory(0),
			target:      2,
			source:      100,
			signingRoot: []byte{1, 2, 3},
			expected:    deprecatedEncodedAttestingHistory{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x64, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x2, 0x3, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
			error:       "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc, err := tt.enc.setTargetData(ctx,
				tt.target,
				&deprecatedHistoryData{
					Source:      tt.source,
					SigningRoot: tt.signingRoot,
				})
			if tt.error == "" {
				require.NoError(t, err)
				td, err := enc.getTargetData(ctx, tt.target)
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
