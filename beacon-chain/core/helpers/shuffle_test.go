package helpers

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
)

func TestShuffleList_InvalidValidatorCount(t *testing.T) {
	maxShuffleListSize = 20
	list := make([]uint64, 21)
	if _, err := ShuffleList(list, [32]byte{123, 125}); err == nil {
		t.Error("Shuffle should have failed when validator count exceeds ModuloBias")
		maxShuffleListSize = 1 << 40
	}
	maxShuffleListSize = 1 << 40
}

func TestShuffleList_OK(t *testing.T) {
	var list1 []uint64
	seed1 := [32]byte{1, 128, 12}
	seed2 := [32]byte{2, 128, 12}
	for i := 0; i < 10; i++ {
		list1 = append(list1, uint64(i))
	}

	list2 := make([]uint64, len(list1))
	copy(list2, list1)

	list1, err := ShuffleList(list1, seed1)
	if err != nil {
		t.Errorf("Shuffle failed with: %v", err)
	}

	list2, err = ShuffleList(list2, seed2)
	if err != nil {
		t.Errorf("Shuffle failed with: %v", err)
	}

	if reflect.DeepEqual(list1, list2) {
		t.Errorf("2 shuffled lists shouldn't be equal")
	}
	if !reflect.DeepEqual(list1, []uint64{0, 7, 8, 6, 3, 9, 4, 5, 2, 1}) {
		t.Errorf("list 1 was incorrectly shuffled got: %v", list1)
	}
	if !reflect.DeepEqual(list2, []uint64{0, 5, 2, 1, 6, 8, 7, 3, 4, 9}) {
		t.Errorf("list 2 was incorrectly shuffled got: %v", list2)
	}
}

func TestSplitIndices_OK(t *testing.T) {
	var l []uint64
	numValidators := uint64(64000)
	for i := uint64(0); i < numValidators; i++ {
		l = append(l, i)
	}
	split := SplitIndices(l, params.BeaconConfig().SlotsPerEpoch)
	if uint64(len(split)) != params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Split list failed due to incorrect length, wanted:%v, got:%v", params.BeaconConfig().SlotsPerEpoch, len(split))
	}

	for _, s := range split {
		if uint64(len(s)) != numValidators/params.BeaconConfig().SlotsPerEpoch {
			t.Errorf("Split list failed due to incorrect length, wanted:%v, got:%v", numValidators/params.BeaconConfig().SlotsPerEpoch, len(s))
		}
	}
}

func TestShuffleList_Vs_ShuffleIndex(t *testing.T) {
	list := []uint64{}
	listSize := uint64(1000)
	seed := [32]byte{123, 42}
	for i := uint64(0); i < listSize; i++ {
		list = append(list, i)
	}
	shuffledListByIndex := make([]uint64, listSize)
	for i := uint64(0); i < listSize; i++ {
		si, err := ShuffledIndex(i, listSize, seed)
		if err != nil {
			t.Error(err)
		}
		shuffledListByIndex[si] = i
	}
	shuffledList, err := ShuffleList(list, seed)
	if err != nil {
		t.Fatalf("shuffled list error: %v", err)

	}
	if !reflect.DeepEqual(shuffledList, shuffledListByIndex) {
		t.Errorf("shuffled lists ar not equal shuffled list: %v shuffled list by index: %v", shuffledList, shuffledListByIndex)
	}

}

func BenchmarkShuffledIndex(b *testing.B) {
	listSizes := []uint64{4000000, 40000, 400}
	seed := [32]byte{123, 42}
	for _, listSize := range listSizes {
		b.Run(fmt.Sprintf("ShuffledIndex_%d", listSize), func(ib *testing.B) {
			for i := uint64(0); i < uint64(ib.N); i++ {
				if _, err := ShuffledIndex(i%listSize, listSize, seed); err != nil {
					b.Error(err)
				}
			}
		})
	}
}

func BenchmarkIndexComparison(b *testing.B) {
	listSizes := []uint64{400000, 40000, 400}
	seed := [32]byte{123, 42}
	for _, listSize := range listSizes {
		b.Run(fmt.Sprintf("Indexwise_ShuffleList_%d", listSize), func(ib *testing.B) {
			for i := 0; i < ib.N; i++ {
				// Simulate a list-shuffle by running shuffle-index listSize times.
				for j := uint64(0); j < listSize; j++ {
					if _, err := ShuffledIndex(j, listSize, seed); err != nil {
						b.Error(err)
					}
				}
			}
		})
	}
}

func BenchmarkShuffleList(b *testing.B) {
	listSizes := []uint64{400000, 40000, 400}
	seed := [32]byte{123, 42}
	for _, listSize := range listSizes {
		testIndices := make([]uint64, listSize)
		for i := uint64(0); i < listSize; i++ {
			testIndices[i] = i
		}
		b.Run(fmt.Sprintf("ShuffleList_%d", listSize), func(ib *testing.B) {
			for i := 0; i < ib.N; i++ {
				if _, err := ShuffleList(testIndices, seed); err != nil {
					b.Error(err)
				}
			}
		})
	}
}

func TestShuffledIndex(t *testing.T) {
	list := []uint64{}
	listSize := uint64(399)
	for i := uint64(0); i < listSize; i++ {
		list = append(list, i)
	}
	shuffledList := make([]uint64, listSize)
	unShuffledList := make([]uint64, listSize)
	seed := [32]byte{123, 42}
	for i := uint64(0); i < listSize; i++ {
		si, err := ShuffledIndex(i, listSize, seed)
		if err != nil {
			t.Error(err)
		}
		shuffledList[si] = i
	}
	for i := uint64(0); i < listSize; i++ {
		ui, err := UnShuffledIndex(i, listSize, seed)
		if err != nil {
			t.Error(err)
		}
		unShuffledList[ui] = shuffledList[i]
	}
	if !reflect.DeepEqual(unShuffledList, list) {
		t.Errorf("Want: %v got: %v", list, unShuffledList)
	}

}

func TestSplitIndicesAndOffset_OK(t *testing.T) {
	var l []uint64
	validators := uint64(64000)
	for i := uint64(0); i < validators; i++ {
		l = append(l, i)
	}
	chunks := uint64(6)
	split := SplitIndices(l, chunks)
	for i := uint64(0); i < chunks; i++ {
		if !reflect.DeepEqual(split[i], l[sliceutil.SplitOffset(uint64(len(l)), chunks, i):sliceutil.SplitOffset(uint64(len(l)), chunks, i+1)]) {
			t.Errorf("Want: %v got: %v", l[sliceutil.SplitOffset(uint64(len(l)), chunks, i):sliceutil.SplitOffset(uint64(len(l)), chunks, i+1)], split[i])
			break
		}
	}
}
