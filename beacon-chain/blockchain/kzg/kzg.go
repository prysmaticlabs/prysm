package kzg

import (
	ckzg4844 "github.com/ethereum/c-kzg-4844/bindings/go"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
)

// BytesPerBlob is the number of bytes in a single blob.
const BytesPerBlob = ckzg4844.BytesPerBlob

// Blob represents a serialized chunk of data.
type Blob [BytesPerBlob]byte

// BytesPerCell is the number of bytes in a single cell.
const BytesPerCell = ckzg4844.BytesPerCell

// Cell represents a chunk of an encoded Blob.
type Cell [BytesPerCell]byte

// Commitment represent a KZG commitment to a Blob.
type Commitment [48]byte

// Proof represents a KZG proof that attests to the validity of a Blob or parts of it.
type Proof [48]byte

// Bytes48 is a 48-byte array.
type Bytes48 = ckzg4844.Bytes48

// Bytes32 is a 32-byte array.
type Bytes32 = ckzg4844.Bytes32

// CellsAndProofs represents the Cells and Proofs corresponding to
// a single blob.

type CellsAndProofs struct {
	Cells  []Cell
	Proofs []Proof
}

// Set to false to not change any existing behavior
const USE_GO_ETH_KZG = true

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

func ComputeCellsAndKZGProofs(blob *Blob) (CellsAndProofs, error) {
	if USE_GO_ETH_KZG {
		return computeCellsAndKZGProofsGoEthKZG(blob)
	} else {
		return computeCellsAndKZGProofscKZG(blob)
	}
}

func VerifyCellKZGProofBatch(commitmentsBytes []Bytes48, cellIndices []uint64, cells []Cell, proofsBytes []Bytes48) (bool, error) {
	if USE_GO_ETH_KZG {
		return verifyCellKZGProofBatchGoEthKZG(commitmentsBytes, cellIndices, cells, proofsBytes)
	} else {
		return verifyCellKZGProofBatchcKZG(commitmentsBytes, cellIndices, cells, proofsBytes)
	}
}

func RecoverCellsAndKZGProofs(cellIndices []uint64, partialCells []Cell) (CellsAndProofs, error) {
	if USE_GO_ETH_KZG {
		return recoverCellsAndKZGProofsGoEthKZG(cellIndices, partialCells)
	} else {
		return recoverCellsAndKZGProofscKZG(cellIndices, partialCells)
	}
}
