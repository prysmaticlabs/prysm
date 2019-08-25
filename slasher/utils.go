// Package slasher implements slashing validation
// and proof creation.
package slasher

import (
	"sort"
)

func insertSort(data []uint64, element uint64) []uint64 {
	index := sort.Search(len(data), func(i int) bool { return data[i] > element })
	data = append(data, uint64(0))
	copy(data[index+1:], data[index:])
	data[index] = element
	return data
}

func truncateItems(data []uint64, minItemVal uint64) (truncate bool, truncatedList []uint64, itemsToTruncate []uint64) {
	index := sort.Search(len(data), func(i int) bool { return data[i] > minItemVal })
	if index == 0 {
		return false, data, []uint64{}
	}
	itemsToTruncate = data[:index]
	truncatedList = data[index:]
	return true, truncatedList, itemsToTruncate
}
