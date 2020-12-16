package kv

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func Test_markUnattestedEpochsCorrectly(t *testing.T) {
	ctx := context.Background()
	// First we create an EncHistoryData slice that has certain epochs
	// marked as attested while epochs in between those as empty:
	// (source: 0, signing root: 0x0).
	h1 := NewAttestationHistoryArray(0)
	sr1 := [32]byte{}
	copy(sr1[:], "1")
	h2, err := oldSetHistoryAtTarget(h1, 0, &HistoryData{
		Source:      0,
		SigningRoot: sr1[:],
	})
	require.NoError(t, err)

	// We mark target 50, source 49 as attested.
	sr2 := [32]byte{}
	copy(sr2[:], "50")
	highestEpoch := uint64(50)
	h3, err := oldSetHistoryAtTarget(h2, highestEpoch, &HistoryData{
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

	// Now, we check the epochs we expect have source: 0 and signing root: 0x0
	for i := uint64(2); i < highestEpoch; i++ {
		data, err := h3.GetTargetData(ctx, i)
		require.NoError(t, err)
		require.Equal(t, false, data.IsEmpty())
	}

	// Finally, we ensure the history for those epochs indeed return IsEmpty() == true.
	newHist, err := markUnattestedEpochsCorrectly(h3)
	require.NoError(t, err)
	for i := uint64(2); i < highestEpoch; i++ {
		data, err := newHist.GetTargetData(ctx, i)
		require.NoError(t, err)
		require.Equal(t, true, data.IsEmpty())
	}
}

func oldSetHistoryAtTarget(hd EncHistoryData, target uint64, historyData *HistoryData) (EncHistoryData, error) {
	if err := hd.assertSize(); err != nil {
		return nil, err
	}
	// Cursor for the location to write target epoch to.
	// Modulus of target epoch  X weak subjectivity period in order to have maximum size to the encapsulated data array.
	cursor := latestEpochWrittenSize + (target%params.BeaconConfig().WeakSubjectivityPeriod)*historySize

	if uint64(len(hd)) < cursor+historySize {
		ext := make([]byte, cursor+historySize-uint64(len(hd)))
		hd = append(hd, ext...)
	}
	copy(hd[cursor:cursor+sourceSize], bytesutil.Uint64ToBytesLittleEndian(historyData.Source))
	copy(hd[cursor+sourceSize:cursor+sourceSize+signingRootSize], historyData.SigningRoot)

	return hd, nil
}
