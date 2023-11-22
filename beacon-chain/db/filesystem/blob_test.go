package filesystem

import (
	"bytes"
	"testing"

	ssz "github.com/prysmaticlabs/fastssz"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/verification"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/spf13/afero"

	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
)

func TestBlobStorage_SaveBlobData(t *testing.T) {
	_, sidecars := util.GenerateTestDenebBlockWithSidecar(t, [32]byte{}, 1, fieldparams.MaxBlobsPerBlock)
	testSidecars, err := verification.BlobSidecarSliceNoop(sidecars)
	require.NoError(t, err)

	t.Run("no error for duplicate", func(t *testing.T) {
		fs, bs := NewEphemeralBlobStorageWithFs(t)
		existingSidecar := testSidecars[0]

		blobPath := namerForSidecar(existingSidecar).path()
		// Serialize the existing BlobSidecar to binary data.
		existingSidecarData, err := ssz.MarshalSSZ(existingSidecar)
		require.NoError(t, err)

		require.NoError(t, bs.Save(existingSidecar))
		// No error when attempting to write twice.
		require.NoError(t, bs.Save(existingSidecar))

		content, err := afero.ReadFile(fs, blobPath)
		require.NoError(t, err)
		require.Equal(t, true, bytes.Equal(existingSidecarData, content))

		// Deserialize the BlobSidecar from the saved file data.
		savedSidecar := &ethpb.BlobSidecar{}
		err = savedSidecar.UnmarshalSSZ(content)
		require.NoError(t, err)

		// Compare the original Sidecar and the saved Sidecar.
		require.DeepSSZEqual(t, existingSidecar.BlobSidecar, savedSidecar)

	})
	t.Run("indices", func(t *testing.T) {
		bs := NewEphemeralBlobStorage(t)
		sc := testSidecars[2]
		require.NoError(t, bs.Save(sc))
		actualSc, err := bs.Get(sc.BlockRoot(), sc.Index)
		require.NoError(t, err)
		expectedIdx := [fieldparams.MaxBlobsPerBlock]bool{false, false, true}
		actualIdx, err := bs.Indices(actualSc.BlockRoot())
		require.NoError(t, err)
		require.Equal(t, expectedIdx, actualIdx)
	})

	t.Run("round trip write then read", func(t *testing.T) {
		bs := NewEphemeralBlobStorage(t)
		err := bs.Save(testSidecars[0])
		require.NoError(t, err)

		expected := testSidecars[0]
		actual, err := bs.Get(expected.BlockRoot(), expected.Index)
		require.NoError(t, err)
		require.DeepSSZEqual(t, expected, actual)
	})
}

func TestBlobIndicesBounds(t *testing.T) {
	fs, bs := NewEphemeralBlobStorageWithFs(t)
	root := [32]byte{}

	okIdx := uint64(fieldparams.MaxBlobsPerBlock - 1)
	writeFakeSSZ(t, fs, root, okIdx)
	indices, err := bs.Indices(root)
	require.NoError(t, err)
	var expected [fieldparams.MaxBlobsPerBlock]bool
	expected[okIdx] = true
	for i := range expected {
		require.Equal(t, expected[i], indices[i])
	}

	oobIdx := uint64(fieldparams.MaxBlobsPerBlock)
	writeFakeSSZ(t, fs, root, oobIdx)
	_, err = bs.Indices(root)
	require.ErrorIs(t, err, errIndexOutOfBounds)
}

func writeFakeSSZ(t *testing.T, fs afero.Fs, root [32]byte, idx uint64) {
	namer := blobNamer{root: root, index: idx}
	require.NoError(t, fs.MkdirAll(namer.dir(), 0700))
	fh, err := fs.Create(namer.path())
	require.NoError(t, err)
	_, err = fh.Write([]byte("derp"))
	require.NoError(t, err)
	require.NoError(t, fh.Close())
}
