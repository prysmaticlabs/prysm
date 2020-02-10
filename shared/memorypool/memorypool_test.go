package memorypool

import (
	"testing"
)

func TestRoundTripMemoryRetrieval(t *testing.T) {
	byteSlice := make([][]byte, 1000)
	PutDoubleByteSlice(byteSlice)
	newSlice := GetDoubleByteSlice(1000)

	if len(newSlice) != 1000 {
		t.Errorf("Wanted same slice object, but got different object. "+
			"Wanted  slice with length %d but got length %d", 1000, len(newSlice))
	}
}
