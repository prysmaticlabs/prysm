package utils

import (
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestPermutedIndex(t *testing.T) {
	hash1 := common.BytesToHash([]byte{'a', 'b', 'c', 'd', 'e', 'f', 'g', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'a', 'b', 'c', 'd', 'e', 'f', 'g'})
	hash2 := common.BytesToHash([]byte{'1', '2', '3', '4', '5', '6', '7', '1', '2', '3', '4', '5', '6', '7', '1', '2', '3', '4', '5', '6', '7', '1', '2', '3', '4', '5', '6', '7', '1', '2', '3', '4', '5', '6', '7'})
	var list1 []uint64

	for i := 0; i < 10; i++ {
		list1 = append(list1, uint64(i))
	}

	list2 := make([]uint64, len(list1))
	copy(list2, list1)

	for i, v := range list1 {
		indice, err := PermutedIndex(v, uint64(len(list1)), hash1)
		if err != nil {
			t.Errorf("Shuffle failed with: %v", err)
		}
		list1[i] = indice
	}

	for i, v := range list2 {
		indice, err := PermutedIndex(v, uint64(len(list2)), hash2)
		if err != nil {
			t.Errorf("Shuffle failed with: %v", err)
		}
		list2[i] = indice
	}

	if reflect.DeepEqual(list1, list2) {
		t.Errorf("2 shuffled lists shouldn't be equal list1: %v , list2: %v", list1, list2)
	}
	expected1 := []uint64{6, 3, 2, 5, 8, 0, 4, 7, 1, 9}
	if !reflect.DeepEqual(list1, expected1) {
		t.Errorf("list 1 was incorrectly shuffled. Expeced: %v, Received: %v", expected1, list1)
	}
	expected2 := []uint64{6, 2, 0, 9, 4, 1, 8, 5, 3, 7}
	if !reflect.DeepEqual(list2, expected2) {
		t.Errorf("list 2 was incorrectly shuffled. Expected: %v, Received: %v", expected2, list2)
	}

}

func TestSplitIndices_OK(t *testing.T) {
	var l []uint64
	validators := 64000
	for i := 0; i < validators; i++ {
		l = append(l, uint64(i))
	}
	split := SplitIndices(l, params.BeaconConfig().SlotsPerEpoch)
	if len(split) != int(params.BeaconConfig().SlotsPerEpoch) {
		t.Errorf("Split list failed due to incorrect length, wanted:%v, got:%v", params.BeaconConfig().SlotsPerEpoch, len(split))
	}

	for _, s := range split {
		if len(s) != validators/int(params.BeaconConfig().SlotsPerEpoch) {
			t.Errorf("Split list failed due to incorrect length, wanted:%v, got:%v", validators/int(params.BeaconConfig().SlotsPerEpoch), len(s))
		}
	}
}

func BenchmarkPermutedIndex(b *testing.B) {
	hash1 := common.BytesToHash([]byte{'a', 'b', 'c', 'd', 'e', 'f', 'g', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'a', 'b', 'c', 'd', 'e', 'f', 'g'})
	list1 := make([]uint64, 10)
	for i := uint64(0); i < 10; i++ {
		list1[i] = i
	}

	b.N = 50
	b.ReportAllocs()
	b.ResetTimer()
	for i, v := range list1 {
		indice, err := PermutedIndex(v, uint64(len(list1)), hash1)
		if err != nil {
			b.Errorf("Shuffle failed with: %v", err)
		}
		list1[i] = indice
	}
}
