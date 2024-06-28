package kzg

import (
	"errors"

	GoKZG "github.com/crate-crypto/go-kzg-4844"
	ckzg4844 "github.com/ethereum/c-kzg-4844/bindings/go"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
)

// Blob represents a serialized chunk of data.
type Blob [131072]byte

// Commitment represent a KZG commitment to a Blob.
type Commitment [48]byte

// Proof represents a KZG proof that attests to the validity of a Blob or parts of it.
type Proof [48]byte

// Bytes48 is a 48-byte array.
type Bytes48 = ckzg4844.Bytes48

// Bytes32 is a 32-byte array.
type Bytes32 = ckzg4844.Bytes32

// BytesPerCell is the number of bytes in a single cell.
const BytesPerCell = ckzg4844.FieldElementsPerCell * ckzg4844.BytesPerFieldElement

// BytesPerBlob is the number of bytes in a single blob.
const BytesPerBlob = ckzg4844.BytesPerBlob

// FieldElementsPerCell is the number of field elements in a single cell.
// TODO: This should not be exposed.
const FieldElementsPerCell = ckzg4844.FieldElementsPerCell

// CellsPerExtBlob is the number of cells that we generate for a single blob.
// This is equivalent to the number of columns in the data matrix.
const CellsPerExtBlob = ckzg4844.CellsPerExtBlob

// Cell represents a chunk of an encoded Blob.
// TODO: This is not correctly sized in c-kzg
// TODO: It should be a vector of bytes
// TODO: Note that callers of this package rely on `BytesPerCell`
type Cell ckzg4844.Cell

// Verify performs single or batch verification of commitments depending on the number of given BlobSidecars.
func Verify(sidecars ...blocks.ROBlob) error {
	if len(sidecars) == 0 {
		return nil
	}
	if len(sidecars) == 1 {
		return kzgContext.VerifyBlobKZGProof(
			bytesToBlob(sidecars[0].Blob),
			bytesToCommitment(sidecars[0].KzgCommitment),
			bytesToKZGProof(sidecars[0].KzgProof))
	}
	blobs := make([]GoKZG.Blob, len(sidecars))
	cmts := make([]GoKZG.KZGCommitment, len(sidecars))
	proofs := make([]GoKZG.KZGProof, len(sidecars))
	for i, sidecar := range sidecars {
		blobs[i] = bytesToBlob(sidecar.Blob)
		cmts[i] = bytesToCommitment(sidecar.KzgCommitment)
		proofs[i] = bytesToKZGProof(sidecar.KzgProof)
	}
	return kzgContext.VerifyBlobKZGProofBatch(blobs, cmts, proofs)
}

func BlobToKZGCommitment(blob *Blob) (Commitment, error) {
	comm, err := kzg4844.BlobToCommitment(kzg4844.Blob(*blob))
	if err != nil {
		return Commitment{}, err
	}
	return Commitment(comm), nil
}

func ComputeBlobKZGProof(blob *Blob, commitment Commitment) (Proof, error) {
	proof, err := kzg4844.ComputeBlobProof(kzg4844.Blob(*blob), kzg4844.Commitment(commitment))
	if err != nil {
		return [48]byte{}, err
	}
	return Proof(proof), nil
}

func ComputeCellsAndKZGProofs(blob *Blob) ([ckzg4844.CellsPerExtBlob]Cell, [ckzg4844.CellsPerExtBlob]Proof, error) {
	ckzgBlob := ckzg4844.Blob(*blob)
	_cells, _proofs, err := ckzg4844.ComputeCellsAndKZGProofs(&ckzgBlob)
	if err != nil {
		return [ckzg4844.CellsPerExtBlob]Cell{}, [ckzg4844.CellsPerExtBlob]Proof{}, err
	}

	// Convert Cells and Proofs to types defined in this package
	var cells [ckzg4844.CellsPerExtBlob]Cell
	for i := range _cells {
		cells[i] = Cell(cells[i])
	}

	var proofs [ckzg4844.CellsPerExtBlob]Proof
	for i, proof := range _proofs {
		proofs[i] = Proof(proof)
	}

	return cells, proofs, nil
}

func VerifyCellKZGProof(commitmentBytes Bytes48, cellId uint64, cell Cell, proofBytes Bytes48) (bool, error) {
	return ckzg4844.VerifyCellKZGProof(commitmentBytes, cellId, ckzg4844.Cell(cell), proofBytes)
}

func VerifyCellKZGProofBatch(commitmentsBytes []Bytes48, rowIndices, columnIndices []uint64, cells []Cell, proofsBytes []Bytes48) (bool, error) {
	// Convert `Cell` type to `ckgz4844.Cell`
	var ckzgCells []ckzg4844.Cell
	for i := range cells {
		ckzgCells[i] = ckzg4844.Cell(cells[i])
	}

	return ckzg4844.VerifyCellKZGProofBatch(commitmentsBytes, rowIndices, columnIndices, ckzgCells, proofsBytes)
}

func RecoverAllCells(cellIds []uint64, cells []Cell) ([ckzg4844.CellsPerExtBlob]Cell, error) {
	// Convert `Cell` type to `ckgz4844.Cell`
	var ckzgCells []ckzg4844.Cell
	for i := range cells {
		ckzgCells[i] = ckzg4844.Cell(cells[i])
	}

	recoveredCells, err := ckzg4844.RecoverAllCells(cellIds, ckzgCells)
	if err != nil {
		return [ckzg4844.CellsPerExtBlob]Cell{}, err
	}

	// This should never happen, we return an error instead of panicking.
	if len(recoveredCells) != ckzg4844.CellsPerExtBlob {
		return [ckzg4844.CellsPerExtBlob]Cell{}, errors.New("recovered cells length is not equal to CellsPerExtBlob")
	}

	// Convert `ckgz4844.Cell` type to `Cell`
	var ret [ckzg4844.CellsPerExtBlob]Cell
	for i := range recoveredCells {
		ret[i] = Cell(recoveredCells[i])
	}
	return ret, nil
}

func CellsToBlob(cells [ckzg4844.CellsPerExtBlob]Cell) (Blob, error) {
	// Convert `Cell` type to `ckgz4844.Cell`
	var ckzgCells [ckzg4844.CellsPerExtBlob]ckzg4844.Cell
	for i := range cells {
		ckzgCells[i] = ckzg4844.Cell(cells[i])
	}

	blob, err := ckzg4844.CellsToBlob(ckzgCells)
	if err != nil {
		return Blob{}, err
	}

	return Blob(blob), nil
}

func bytesToBlob(blob []byte) (ret GoKZG.Blob) {
	copy(ret[:], blob)
	return
}

func bytesToCommitment(commitment []byte) (ret GoKZG.KZGCommitment) {
	copy(ret[:], commitment)
	return
}

func bytesToKZGProof(proof []byte) (ret GoKZG.KZGProof) {
	copy(ret[:], proof)
	return
}
