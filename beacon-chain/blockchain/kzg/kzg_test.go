package kzg

import "testing"

func TestCellFlattenedChunked(t *testing.T) {
	cell := makeCell()
	chunkedCell := cellToChunkedCell(cell)
	flattenedCell := cellChunkedToCell(chunkedCell)
	if cell != flattenedCell {
		t.Errorf("cell != flattenedCell")
	}
}

func makeCell() Cell {
	var cell Cell
	for i := 0; i < fieldElementsPerCell; i++ {
		rand32 := deterministicRandomness(i)
		copy(cell[i][:], rand32[:])
	}
	return cell
}
