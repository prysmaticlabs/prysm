package filesystem

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	ssz "github.com/prysmaticlabs/fastssz"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/io/file"
	"github.com/prysmaticlabs/prysm/v4/proto/eth/v2"

	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func TestBlobStorage_SaveBlobData(t *testing.T) {
	testSidecars := generateBlobSidecars(t, []primitives.Slot{100}, fieldparams.MaxBlobsPerBlock)
	t.Run("NoBlobData", func(t *testing.T) {
		tempDir := t.TempDir()
		bs := &BlobStorage{baseDir: tempDir}
		err := bs.SaveBlobData([]*eth.BlobSidecar{})
		require.ErrorContains(t, "no blob data to save", err)
	})

	t.Run("BlobExists", func(t *testing.T) {
		tempDir := t.TempDir()
		bs := &BlobStorage{baseDir: tempDir}
		existingSidecar := testSidecars[0]

		blobPath := bs.sidecarFileKey(existingSidecar)
		// Serialize the existing BlobSidecar to binary data.
		existingSidecarData, err := ssz.MarshalSSZ(existingSidecar)
		require.NoError(t, err)

		err = os.MkdirAll(filepath.Dir(blobPath), os.ModePerm)
		require.NoError(t, err)

		// Write the serialized data to the blob file.
		err = os.WriteFile(blobPath, existingSidecarData, os.ModePerm)
		require.NoError(t, err)

		err = bs.SaveBlobData([]*eth.BlobSidecar{existingSidecar})
		require.NoError(t, err)

		content, err := os.ReadFile(blobPath)
		require.NoError(t, err)

		// Deserialize the BlobSidecar from the saved file data.
		var savedSidecar ssz.Unmarshaler
		savedSidecar = &eth.BlobSidecar{}
		err = savedSidecar.UnmarshalSSZ(content)
		require.NoError(t, err)

		// Compare the original Sidecar and the saved Sidecar.
		require.DeepSSZEqual(t, existingSidecar, savedSidecar)
	})

	//t.Run("SaveBlobDataNoErrors", func(t *testing.T) {
	//	tempDir := t.TempDir()
	//	bs := &BlobStorage{baseDir: tempDir}
	//	err := bs.SaveBlobData(testSidecars)
	//	require.NoError(t, err)
	//
	//	// Check the number of block root directories.
	//	blockRootDirs, err := os.ReadDir(tempDir)
	//	require.NoError(t, err)
	//	require.Equal(t, 1, len(blockRootDirs))
	//
	//	for _, blockRootDir := range blockRootDirs {
	//		blockRootPath := filepath.Join(tempDir, blockRootDir.Name())
	//
	//		// Check the number of files in the block root directory.
	//		files, err := os.ReadDir(blockRootPath)
	//		require.NoError(t, err)
	//		require.Equal(t, 7, len(files)) // Assuming there is only one file per block root for this test
	//
	//		content, err := os.ReadFile(filepath.Join(blockRootPath, files[0].Name()))
	//		require.NoError(t, err)
	//
	//		// Deserialize the BlobSidecar from the saved file data.
	//		var savedSidecar ssz.Unmarshaler
	//		savedSidecar = &eth.BlobSidecar{}
	//		err = savedSidecar.UnmarshalSSZ(content)
	//		require.NoError(t, err)
	//
	//		// Find the corresponding test sidecar based on the file name.
	//		sidecar := findTestSidecarsByFileName(t, testSidecars, files[0].Name())
	//		require.NotNil(t, sidecar)
	//		// Compare the original Sidecar and the saved Sidecar.
	//		require.DeepSSZEqual(t, sidecar, savedSidecar)
	//	}
	//})

	t.Run("OverwriteBlobWithDifferentContent", func(t *testing.T) {
		tempDir := t.TempDir()
		bs := &BlobStorage{baseDir: tempDir}
		originalSidecar := []*eth.BlobSidecar{testSidecars[0]}
		// Save the original sidecar
		err := bs.SaveBlobData(originalSidecar)
		require.NoError(t, err)

		// Modify the blob data
		modifiedSidecar := originalSidecar
		modifiedSidecar[0].Blob = []byte("Modified Blob Data")

		err = bs.SaveBlobData(modifiedSidecar)
		require.ErrorContains(t, "failed to save blob sidecar, tried to overwrite blob", err)
	})
}

func findTestSidecarsByFileName(t *testing.T, testSidecars []*eth.BlobSidecar, fileName string) *eth.BlobSidecar {
	parts := strings.SplitN(fileName, ".", 2)
	require.Equal(t, 2, len(parts))
	// parts[0] contains the substring before the first period
	components := strings.Split(parts[0], string(filepath.Separator))
	if len(components) == 2 {
		blobIndex, err := strconv.Atoi(components[1])
		require.NoError(t, err)
		for _, sidecar := range testSidecars {
			if sidecar.Index == uint64(blobIndex) {
				return sidecar
			}
		}
	}
	return nil
}

func TestCheckDataIntegrity(t *testing.T) {
	testSidecars := generateBlobSidecars(t, []primitives.Slot{100}, fieldparams.MaxBlobsPerBlock)
	originalData, err := ssz.MarshalSSZ(testSidecars[0])
	require.NoError(t, err)
	originalChecksum := sha256.Sum256(originalData)

	tempDir := t.TempDir()
	tempfile, err := os.CreateTemp(tempDir, "testfile")
	require.NoError(t, err)
	_, err = tempfile.Write(originalData)
	require.NoError(t, err)

	err = checkDataIntegrity(testSidecars[0], tempfile.Name())
	require.NoError(t, err)

	// Modify the data in the file to simulate data corruption
	corruptedData := []byte("corrupted data")
	err = os.WriteFile(tempfile.Name(), corruptedData, os.ModePerm)
	require.NoError(t, err)

	// Test data integrity check with corrupted data
	err = checkDataIntegrity(testSidecars[0], tempfile.Name())
	require.ErrorContains(t, "data integrity check failed", err)

	// Modify the calculated checksum to be incorrect
	wrongChecksum := hex.EncodeToString(originalChecksum[:]) + "12345"
	err = os.WriteFile(tempfile.Name(), []byte(wrongChecksum), os.ModePerm)
	require.NoError(t, err)

	checksum, err := file.HashFile(tempfile.Name())
	require.NoError(t, err)
	require.NotEqual(t, wrongChecksum, hex.EncodeToString(checksum))
}

func generateBlobSidecars(t *testing.T, slots []primitives.Slot, n uint64) []*eth.BlobSidecar {
	length := n * uint64(len(slots))
	blobSidecars := make([]*eth.BlobSidecar, length)
	index := uint64(0)
	for _, slot := range slots {
		for i := 0; i < int(n); i++ {
			blobSidecars[index] = generateBlobSidecar(t, slot, index, nil)
			index++
		}
	}
	return blobSidecars
}

func generateBlobSidecar(tb testing.TB, slot primitives.Slot, index uint64, root []byte) *eth.BlobSidecar {
	blob := make([]byte, 131072)
	_, err := rand.Read(blob)
	require.NoError(tb, err)
	kzgCommitment := make([]byte, 48)
	_, err = rand.Read(kzgCommitment)
	require.NoError(tb, err)
	kzgProof := make([]byte, 48)
	_, err = rand.Read(kzgProof)
	require.NoError(tb, err)
	if len(root) == 0 {
		root = bytesutil.PadTo(bytesutil.ToBytes(uint64(slot), 32), 32)
	}
	return &eth.BlobSidecar{
		BlockRoot:       root,
		Index:           index,
		Slot:            slot,
		BlockParentRoot: bytesutil.PadTo([]byte{'b'}, 32),
		ProposerIndex:   101,
		Blob:            blob,
		KzgCommitment:   kzgCommitment,
		KzgProof:        kzgProof,
	}
}
