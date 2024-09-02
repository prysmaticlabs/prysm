package kzg

import (
	"errors"

	goethkzg "github.com/crate-crypto/go-eth-kzg"
)

func computeCellsAndKZGProofsGoEthKZG(blob *Blob) (CellsAndProofs, error) {
	goEthKZGBlob := (*goethkzg.Blob)(blob)
	cells, proofs, err := goEthKZGContext.ComputeCellsAndKZGProofs(goEthKZGBlob, 0)
	if err != nil {
		return CellsAndProofs{}, err
	}
	return makeCellsAndProofsGoEthKZG(cells[:], proofs[:])
}

func verifyCellKZGProofBatchGoEthKZG(commitmentsBytes []Bytes48, cellIndices []uint64, cells []Cell, proofsBytes []Bytes48) (bool, error) {
	kzgCommitments := convertBytes48SliceToKZGCommitmentSlice(commitmentsBytes)
	kzgCells := convertCellSliceToPointers(cells)
	kzgProofs := convertBytes48SliceToKZGProofSlice(proofsBytes)

	err := goEthKZGContext.VerifyCellKZGProofBatch(kzgCommitments, cellIndices, kzgCells, kzgProofs)
	if err != nil {
		return false, err
	}
	// TODO: This conforms to the c-kzg API, I think we should change this to only return an error
	return true, nil
}

func recoverCellsAndKZGProofsGoEthKZG(cellIndices []uint64, partialCells []Cell) (CellsAndProofs, error) {
	kzgCells := convertCellSliceToPointers(partialCells)
	cells, proofs, err := goEthKZGContext.RecoverCellsAndComputeKZGProofs(cellIndices, kzgCells, 0)
	if err != nil {
		return CellsAndProofs{}, err
	}

	return makeCellsAndProofsGoEthKZG(cells[:], proofs[:])
}

// Convert c-kzg cells/proofs to the CellsAndProofs type defined in this package.
func makeCellsAndProofsGoEthKZG(goethkzgCells []*goethkzg.Cell, goethkzgProofs []goethkzg.KZGProof) (CellsAndProofs, error) {
	if len(goethkzgCells) != len(goethkzgProofs) {
		return CellsAndProofs{}, errors.New("different number of cells/proofs")
	}

	var cells []Cell
	var proofs []Proof
	for i := range goethkzgCells {
		cells = append(cells, Cell(*goethkzgCells[i]))
		proofs = append(proofs, Proof(goethkzgProofs[i]))
	}

	return CellsAndProofs{
		Cells:  cells,
		Proofs: proofs,
	}, nil
}

func convertBytes48SliceToKZGCommitmentSlice(bytes48Slice []Bytes48) []goethkzg.KZGCommitment {
	commitments := make([]goethkzg.KZGCommitment, len(bytes48Slice))
	for i, b48 := range bytes48Slice {
		copy(commitments[i][:], b48[:])
	}
	return commitments
}

func convertBytes48SliceToKZGProofSlice(bytes48Slice []Bytes48) []goethkzg.KZGProof {
	commitments := make([]goethkzg.KZGProof, len(bytes48Slice))
	for i, b48 := range bytes48Slice {
		copy(commitments[i][:], b48[:])
	}
	return commitments
}

func convertCellSliceToPointers(cells []Cell) []*goethkzg.Cell {
	cellPointers := make([]*goethkzg.Cell, len(cells))
	for i := range cells {
		kzgCell := goethkzg.Cell(cells[i])
		cellPointers[i] = &kzgCell
	}
	return cellPointers
}
