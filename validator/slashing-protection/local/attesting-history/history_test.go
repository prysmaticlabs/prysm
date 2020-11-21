package attestinghistory

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestNew(t *testing.T) {
	ba := New(0)
	assert.Equal(t, latestEpochWrittenSize+historySize, len(ba))
	ba = New(params.BeaconConfig().WeakSubjectivityPeriod - 1)
	assert.Equal(t, latestEpochWrittenSize+historySize*params.BeaconConfig().WeakSubjectivityPeriod, uint64(len(ba)))
	ba = New(params.BeaconConfig().WeakSubjectivityPeriod)
	assert.Equal(t, latestEpochWrittenSize+historySize, len(ba))
	ba = New(params.BeaconConfig().WeakSubjectivityPeriod + 1)
	assert.Equal(t, latestEpochWrittenSize+historySize+historySize, len(ba))
}

func TestSizeChecks(t *testing.T) {
	require.ErrorContains(t, "is smaller then minimal size", assertSize(History{}))
	require.NoError(t, assertSize(History{0, 1, 2, 3, 4, 5, 6, 7}))
	require.ErrorContains(t, "is not a multiple of entry size", assertSize(History{0, 1, 2, 3, 4, 5, 6, 7, 8}))
	require.NoError(t, assertSize(New(0)))
	require.NoError(t, assertSize(New(1)))
	require.NoError(t, assertSize(New(params.BeaconConfig().WeakSubjectivityPeriod)))
	require.NoError(t, assertSize(New(params.BeaconConfig().WeakSubjectivityPeriod-1)))
}

func TestGetLatestEpochWritten(t *testing.T) {
	ha := New(0)
	ha[0] = 28
	lew, err := GetLatestEpochWritten(ha)
	require.NoError(t, err)
	assert.Equal(t, uint64(28), lew)
}

func TestSetLatestEpochWritten(t *testing.T) {
	ha := New(0)
	lew, err := SetLatestEpochWritten(ha, 2828282828)
	require.NoError(t, err)
	bytes := lew[:latestEpochWrittenSize]
	assert.Equal(t, uint64(2828282828), bytesutil.FromBytes8(bytes))
}

func TestHistoricalAttestationAtTargetEpoch(t *testing.T) {
	ha := New(0)
	td, err := HistoricalAttestationAtTargetEpoch(ha, 0)
	require.NoError(t, err)
	assert.DeepEqual(t, &HistoricalAttestation{Source: params.BeaconConfig().FarFutureEpoch, SigningRoot: make([]byte, 32)}, td)
	td, err = HistoricalAttestationAtTargetEpoch(ha, 1)
	require.NoError(t, err)
	require.Equal(t, true, td == nil)
}

func TestMarkAsAttested(t *testing.T) {
	type testStruct struct {
		name        string
		enc         History
		target      uint64
		source      uint64
		signingRoot []byte
		expected    History
		error       string
	}
	tests := []testStruct{
		{
			name:        "empty enc",
			enc:         History{},
			target:      0,
			source:      100,
			signingRoot: []byte{1, 2, 3},
			expected:    (History)(nil),
			error:       "encapsulated data size",
		},
		{
			name:        "new enc",
			enc:         New(0),
			target:      0,
			source:      100,
			signingRoot: []byte{1, 2, 3},
			expected:    History{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x64, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x2, 0x3, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
			error:       "",
		},
		{
			name:        "higher target",
			enc:         New(0),
			target:      2,
			source:      100,
			signingRoot: []byte{1, 2, 3},
			expected:    History{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x64, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x2, 0x3, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
			error:       "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc, err := MarkAsAttested(
				tt.enc,
				&HistoricalAttestation{
					Target:      tt.target,
					Source:      tt.source,
					SigningRoot: tt.signingRoot,
				})
			if tt.error == "" {
				require.NoError(t, err)
				td, err := HistoricalAttestationAtTargetEpoch(enc, tt.target)
				require.NoError(t, err)
				require.NotNil(t, td)
				require.DeepEqual(t, bytesutil.PadTo(tt.signingRoot, 32), td.SigningRoot)
				require.Equal(t, tt.source, td.Source)
				return
			}
			assert.ErrorContains(t, tt.error, err)
			require.DeepEqual(t, tt.expected, enc)
		})
	}
}
