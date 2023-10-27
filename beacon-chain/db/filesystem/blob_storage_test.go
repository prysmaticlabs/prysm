package filesystem

import (
	"bytes"
	"encoding/hex"
	"os"
	"path"
	"strings"
	"sync"
	"testing"

	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func TestSaveBlobDataErrors(t *testing.T) {
	bs := &BlobStorage{baseDir: "testdir"}
	t.Run("NoBlobData", func(t *testing.T) {
		err := bs.SaveBlobData([]*ethpb.BlobSidecar{})
		require.ErrorContains(t, "no blob data to save", err)
	})
	t.Run("CreateFileLockError", func(t *testing.T) {
		err := bs.SaveBlobData([]*ethpb.BlobSidecar{{}})
		require.ErrorContains(t, "failed to create file lock", err)
	})
}

func TestSaveBlobDataMultipleSidecars(t *testing.T) {
	tempDir := t.TempDir()
	numGoroutines := 10

	// Use a wait group to synchronize goroutines.
	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bs := &BlobStorage{baseDir: tempDir}
			err := bs.SaveBlobData(testSidecars)
			require.NoError(t, err)
		}()
	}
	// Wait for all goroutines to finish.
	wg.Wait()

	// Check the number of files in the directory.
	files, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	require.Equal(t, len(files), len(testSidecars))

	for _, file := range files {
		content, err := os.ReadFile(path.Join(tempDir, file.Name()))
		require.NoError(t, err)

		// Find the corresponding test sidecar based on the file name.
		sidecar := findTestSidecarsByFileName(t, file.Name())
		require.NotNil(t, sidecar)
		require.Equal(t, string(sidecar.Blob), string(content))
	}
}

func findTestSidecarsByFileName(t *testing.T, fileName string) *ethpb.BlobSidecar {
	components := strings.Split(fileName, "_")
	if len(components) == 4 {
		blockRoot, err := hex.DecodeString(components[1])
		require.NoError(t, err)
		for _, sidecar := range testSidecars {
			if bytes.Equal(sidecar.BlockRoot, blockRoot) {
				return sidecar
			}
		}
	}
	return nil
}

var testSidecars = []*ethpb.BlobSidecar{
	{
		BlockRoot:       []byte{0x01, 0x02, 0x03},
		Index:           1,
		Slot:            12345,
		BlockParentRoot: []byte{0x04, 0x05, 0x06},
		ProposerIndex:   42,
		Blob:            []byte("Test Blob Data 1"),
		KzgCommitment:   []byte{0x0A, 0x0B, 0x0C},
		KzgProof:        []byte("Test Proof 1"),
	},
	{
		BlockRoot:       []byte{0x11, 0x12, 0x13},
		Index:           2,
		Slot:            54321,
		BlockParentRoot: []byte{0x14, 0x15, 0x16},
		ProposerIndex:   58,
		Blob:            []byte("Test Blob Data 2"),
		KzgCommitment:   []byte{0x1A, 0x1B, 0x1C},
		KzgProof:        []byte("Test Proof 2"),
	},
	{
		BlockRoot:       []byte{0x21, 0x22, 0x23},
		Index:           3,
		Slot:            9876,
		BlockParentRoot: []byte{0x24, 0x25, 0x26},
		ProposerIndex:   33,
		Blob:            []byte("Test Blob Data 3"),
		KzgCommitment:   []byte{0x2A, 0x2B, 0x2C},
		KzgProof:        []byte("Test Proof 3"),
	},
	{
		BlockRoot:       []byte{0x31, 0x32, 0x33},
		Index:           4,
		Slot:            13579,
		BlockParentRoot: []byte{0x34, 0x35, 0x36},
		ProposerIndex:   27,
		Blob:            []byte("Test Blob Data 4"),
		KzgCommitment:   []byte{0x3A, 0x3B, 0x3C},
		KzgProof:        []byte("Test Proof 4"),
	},
	{
		BlockRoot:       []byte{0x41, 0x42, 0x43},
		Index:           5,
		Slot:            24680,
		BlockParentRoot: []byte{0x44, 0x45, 0x46},
		ProposerIndex:   77,
		Blob:            []byte("Test Blob Data 5"),
		KzgCommitment:   []byte{0x4A, 0x4B, 0x4C},
		KzgProof:        []byte("Test Proof 5"),
	},
	{
		BlockRoot:       []byte{0x51, 0x52, 0x53},
		Index:           6,
		Slot:            86420,
		BlockParentRoot: []byte{0x54, 0x55, 0x56},
		ProposerIndex:   62,
		Blob:            []byte("Test Blob Data 6"),
		KzgCommitment:   []byte{0x5A, 0x5B, 0x5C},
		KzgProof:        []byte("Test Proof 6"),
	},
	{
		BlockRoot:       []byte{0x61, 0x62, 0x63},
		Index:           7,
		Slot:            19876,
		BlockParentRoot: []byte{0x64, 0x65, 0x66},
		ProposerIndex:   81,
		Blob:            []byte("Test Blob Data 7"),
		KzgCommitment:   []byte{0x6A, 0x6B, 0x6C},
		KzgProof:        []byte("Test Proof 7"),
	},
	{
		BlockRoot:       []byte{0x71, 0x72, 0x73},
		Index:           8,
		Slot:            11111,
		BlockParentRoot: []byte{0x74, 0x75, 0x76},
		ProposerIndex:   99,
		Blob:            []byte("Test Blob Data 8"),
		KzgCommitment:   []byte{0x7A, 0x7B, 0x7C},
		KzgProof:        []byte("Test Proof 8"),
	},
	{
		BlockRoot:       []byte{0x81, 0x82, 0x83},
		Index:           9,
		Slot:            22222,
		BlockParentRoot: []byte{0x84, 0x85, 0x86},
		ProposerIndex:   11,
		Blob:            []byte("Test Blob Data 9"),
		KzgCommitment:   []byte{0x8A, 0x8B, 0x8C},
		KzgProof:        []byte("Test Proof 9"),
	},
	{
		BlockRoot:       []byte{0x91, 0x92, 0x93},
		Index:           10,
		Slot:            33333,
		BlockParentRoot: []byte{0x94, 0x95, 0x96},
		ProposerIndex:   21,
		Blob:            []byte("Test Blob Data 10"),
		KzgCommitment:   []byte{0x9A, 0x9B, 0x9C},
		KzgProof:        []byte("Test Proof 10"),
	},
}
