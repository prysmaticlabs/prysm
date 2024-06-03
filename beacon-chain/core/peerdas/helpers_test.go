package peerdas_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/rand"
	"testing"

	"github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
	GoKZG "github.com/crate-crypto/go-kzg-4844"
	ckzg4844 "github.com/ethereum/c-kzg-4844/bindings/go"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/sirupsen/logrus"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/blockchain/kzg"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/peerdas"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func deterministicRandomness(seed int64) [32]byte {
	// Converts an int64 to a byte slice
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, seed)
	if err != nil {
		logrus.WithError(err).Error("Failed to write int64 to bytes buffer")
		return [32]byte{}
	}
	bytes := buf.Bytes()

	return sha256.Sum256(bytes)
}

// Returns a serialized random field element in big-endian
func getRandFieldElement(seed int64) [32]byte {
	bytes := deterministicRandomness(seed)
	var r fr.Element
	r.SetBytes(bytes[:])

	return GoKZG.SerializeScalar(r)
}

// Returns a random blob using the passed seed as entropy
func getRandBlob(seed int64) ckzg4844.Blob {
	var blob ckzg4844.Blob
	bytesPerBlob := GoKZG.ScalarsPerBlob * GoKZG.SerializedScalarSize
	for i := 0; i < bytesPerBlob; i += GoKZG.SerializedScalarSize {
		fieldElementBytes := getRandFieldElement(seed + int64(i))
		copy(blob[i:i+GoKZG.SerializedScalarSize], fieldElementBytes[:])
	}
	return blob
}

func generateCommitmentAndProof(blob ckzg4844.Blob) (ckzg4844.KZGCommitment, ckzg4844.KZGProof, error) {
	commitment, err := ckzg4844.BlobToKZGCommitment(&blob)
	if err != nil {
		return ckzg4844.KZGCommitment{}, ckzg4844.KZGProof{}, err
	}
	proof, err := ckzg4844.ComputeBlobKZGProof(&blob, ckzg4844.Bytes48(commitment))
	if err != nil {
		return ckzg4844.KZGCommitment{}, ckzg4844.KZGProof{}, err
	}
	return commitment, proof, err
}

func isSubMap(sub, main map[uint64]bool) bool {
	for k, _ := range sub {
		if !main[k] {
			return false
		}
	}

	return true
}

// since we're distributing data via columns, try to construct cell coordinates with all cells in a column
// we can also randomly select cells from each rows to test recovery later.
func randCellCoordinatesForRecover(matrix peerdas.ExtendedMatrix, blobCount, numOfCols uint64) map[peerdas.CellCoordinate]ckzg4844.Cell {
	random := make([]int, numOfCols)
	for i := uint64(0); i < numOfCols; i++ {
		random[i] = rand.Intn(ckzg4844.CellsPerExtBlob)
	}

	cols := make([]uint64, ckzg4844.CellsPerExtBlob)
	for i := uint64(0); i < ckzg4844.CellsPerExtBlob; i++ {
		cols[i] = i
	}
	rand.Shuffle(len(cols), func(i, j int) {
		cols[i], cols[j] = cols[j], cols[i]
	})
	// select 64 random columns
	selectedCols := cols[:numOfCols]

	res := make(map[peerdas.CellCoordinate]ckzg4844.Cell)
	for i := uint64(0); i < blobCount; i++ {
		for j := uint64(0); j < numOfCols; j++ {
			cord := peerdas.CellCoordinate{
				BlobIndex: i,
				CellID:    selectedCols[j],
			}
			res[cord] = matrix[i*ckzg4844.CellsPerExtBlob+selectedCols[j]]
		}
	}

	return res
}

func TestCustodyColumns(t *testing.T) {
	enodeID := enode.HexID("46337140c6f6daffdf7bc7b182355ff942d1b2564e9e8755de4ec4557c4910d9")

	_, err := peerdas.CustodyColumnSubnets(enodeID, params.BeaconConfig().DataColumnSidecarSubnetCount+1)
	require.ErrorContains(t, "custody subnet count larger than data column sidecar subnet count", err)

	// make sure the list is extendable instead of being an entirely new shuffle.
	var prevCols map[uint64]bool
	for i := uint64(0); i < params.BeaconConfig().DataColumnSidecarSubnetCount; i++ {
		columns, err := peerdas.CustodyColumns(enodeID, i)
		require.NoError(t, err)
		require.Equal(t, isSubMap(prevCols, columns), true)
		prevCols = columns
	}

	// make sure the list is deterministic with the same input.
	for i := uint64(0); i < params.BeaconConfig().DataColumnSidecarSubnetCount; i++ {
		columns1, err := peerdas.CustodyColumns(enodeID, i)
		require.NoError(t, err)
		columns2, err := peerdas.CustodyColumns(enodeID, i)
		require.NoError(t, err)
		require.DeepEqual(t, columns1, columns2)
	}
}

func TestCustodyColumnSubnets(t *testing.T) {
	enodeID := enode.HexID("46337140c6f6daffdf7bc7b182355ff942d1b2564e9e8755de4ec4557c4910d9")

	_, err := peerdas.CustodyColumnSubnets(enodeID, params.BeaconConfig().DataColumnSidecarSubnetCount+1)
	require.ErrorContains(t, "custody subnet count larger than data column sidecar subnet count", err)

	// make sure the list is extendable instead of being an entirely new shuffle.
	var prevColumnSubnets map[uint64]bool
	for i := uint64(0); i < params.BeaconConfig().DataColumnSidecarSubnetCount; i++ {
		columnSubnets, err := peerdas.CustodyColumnSubnets(enodeID, i)
		require.NoError(t, err)
		require.Equal(t, isSubMap(prevColumnSubnets, columnSubnets), true)
		prevColumnSubnets = columnSubnets
	}

	// make sure the list is deterministic with the same input.
	for i := uint64(0); i < params.BeaconConfig().DataColumnSidecarSubnetCount; i++ {
		columnSubnets1, err := peerdas.CustodyColumnSubnets(enodeID, i)
		require.NoError(t, err)
		columnSubnets2, err := peerdas.CustodyColumnSubnets(enodeID, i)
		require.NoError(t, err)
		require.DeepEqual(t, columnSubnets1, columnSubnets2)
	}
}

func TestComputeExtendedMatrix(t *testing.T) {
	blobs := []ckzg4844.Blob{
		getRandBlob(0),
		getRandBlob(1),
	}

	matrix, err := peerdas.ComputeExtendedMatrix(blobs)
	require.NoError(t, err)
	require.Equal(t, len(matrix), len(blobs)*ckzg4844.CellsPerExtBlob)
}

func TestRecoverMatrix(t *testing.T) {
	blobs := []ckzg4844.Blob{
		getRandBlob(0),
		getRandBlob(1),
	}

	matrix, err := peerdas.ComputeExtendedMatrix(blobs)
	require.NoError(t, err)

	// cannot recover when column number is less than half of the total columns
	cellCoordinates1 := randCellCoordinatesForRecover(matrix, uint64(len(blobs)), ckzg4844.CellsPerExtBlob/2-1)
	_, err = peerdas.RecoverMatrix(cellCoordinates1, uint64(len(blobs)))
	require.ErrorContains(t, "bad arguments", err)

	// recover when column number is equal or greater than half of the total columns
	cellCoordinates2 := randCellCoordinatesForRecover(matrix, uint64(len(blobs)), ckzg4844.CellsPerExtBlob/2)
	recoveredMatrix, err := peerdas.RecoverMatrix(cellCoordinates2, uint64(len(blobs)))
	require.NoError(t, err)
	require.DeepEqual(t, matrix, recoveredMatrix)
}

func TestVerifyDataColumnSidecarKZGProofs(t *testing.T) {
	dbBlock := util.NewBeaconBlockDeneb()
	require.NoError(t, kzg.Start())

	comms := [][]byte{}
	blobs := []ckzg4844.Blob{}
	for i := int64(0); i < 6; i++ {
		blob := getRandBlob(i)
		commitment, _, err := generateCommitmentAndProof(blob)
		require.NoError(t, err)
		comms = append(comms, commitment[:])
		blobs = append(blobs, blob)
	}

	dbBlock.Block.Body.BlobKzgCommitments = comms
	sBlock, err := blocks.NewSignedBeaconBlock(dbBlock)
	require.NoError(t, err)
	sCars, err := peerdas.DataColumnSidecars(sBlock, blobs)
	require.NoError(t, err)

	for i, sidecar := range sCars {
		verified, err := peerdas.VerifyDataColumnSidecarKZGProofs(sidecar)
		require.NoError(t, err)
		require.Equal(t, true, verified, fmt.Sprintf("sidecar %d failed", i))
	}
}
