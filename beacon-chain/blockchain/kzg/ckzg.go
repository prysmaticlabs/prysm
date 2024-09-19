package kzg

import (
	"errors"

	ckzg4844 "github.com/ethereum/c-kzg-4844/bindings/go"
)

func computeCellsAndKZGProofscKZG(blob *Blob) (CellsAndProofs, error) {
	ckzgBlob := (*ckzg4844.Blob)(blob)
	ckzgCells, ckzgProofs, err := ckzg4844.ComputeCellsAndKZGProofs(ckzgBlob)
	if err != nil {
		return CellsAndProofs{}, err
	}

	return makeCellsAndProofscKZG(ckzgCells[:], ckzgProofs[:])
}

func verifyCellKZGProofBatchcKZG(commitmentsBytes []Bytes48, cellIndices []uint64, cells []Cell, proofsBytes []Bytes48) (bool, error) {
	// Convert `Cell` type to `ckzg4844.Cell`
	ckzgCells := make([]ckzg4844.Cell, len(cells))
	for i := range cells {
		ckzgCells[i] = ckzg4844.Cell(cells[i])
	}

	return ckzg4844.VerifyCellKZGProofBatch(commitmentsBytes, cellIndices, ckzgCells, proofsBytes)
}

func recoverCellsAndKZGProofscKZG(cellIndices []uint64, partialCells []Cell) (CellsAndProofs, error) {
	// Convert `Cell` type to `ckzg4844.Cell`
	ckzgPartialCells := make([]ckzg4844.Cell, len(partialCells))
	for i := range partialCells {
		ckzgPartialCells[i] = ckzg4844.Cell(partialCells[i])
	}

	ckzgCells, ckzgProofs, err := ckzg4844.RecoverCellsAndKZGProofs(cellIndices, ckzgPartialCells)
	if err != nil {
		return CellsAndProofs{}, err
	}

	return makeCellsAndProofscKZG(ckzgCells[:], ckzgProofs[:])
}

// Convert c-kzg cells/proofs to the CellsAndProofs type defined in this package.
func makeCellsAndProofscKZG(ckzgCells []ckzg4844.Cell, ckzgProofs []ckzg4844.KZGProof) (CellsAndProofs, error) {
	if len(ckzgCells) != len(ckzgProofs) {
		return CellsAndProofs{}, errors.New("different number of cells/proofs")
	}

	var cells []Cell
	var proofs []Proof
	for i := range ckzgCells {
		cells = append(cells, Cell(ckzgCells[i]))
		proofs = append(proofs, Proof(ckzgProofs[i]))
	}

	return CellsAndProofs{
		Cells:  cells,
		Proofs: proofs,
	}, nil
}
