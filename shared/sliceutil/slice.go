package sliceutil

import (
	"strings"
)

// SubsetUint64 returns true if the first array is
// completely contained in the second array with time
// complexity of approximately o(n).
func SubsetUint64(a []uint64, b []uint64) bool {
	if len(a) > len(b) {
		return false
	}

	set := make(map[uint64]uint64)
	for _, v := range b {
		set[v]++
	}

	for _, v := range a {
		if count, found := set[v]; !found {
			return false
		} else if count < 1 {
			return false
		} else {
			set[v] = count - 1
		}
	}
	return true
}

// IntersectionUint64 of any number of uint64 slices with time
// complexity of approximately O(n) leveraging a map to
// check for element existence off by a constant factor
// of underlying map efficiency.
func IntersectionUint64(s ...[]uint64) []uint64 {
	if len(s) == 0 {
		return []uint64{}
	}
	if len(s) == 1 {
		return s[0]
	}
	intersect := make([]uint64, 0)
	for i := 1; i < len(s); i++ {
		m := make(map[uint64]bool)
		for j := 0; j < len(s[i-1]); j++ {
			m[s[i-1][j]] = true
		}
		for j := 0; j < len(s[i]); j++ {
			if _, found := m[s[i][j]]; found {
				intersect = append(intersect, s[i][j])
			}
		}
	}
	return intersect
}

// UnionUint64 of any number of uint64 slices with time
// complexity of approximately O(n) leveraging a map to
// check for element existence off by a constant factor
// of underlying map efficiency.
func UnionUint64(s ...[]uint64) []uint64 {
	if len(s) == 0 {
		return []uint64{}
	}
	if len(s) == 1 {
		return s[0]
	}
	set := s[0]
	m := make(map[uint64]bool)
	for i := 1; i < len(s); i++ {
		a := s[i-1]
		b := s[i]
		for j := 0; j < len(a); j++ {
			m[a[j]] = true
		}
		for j := 0; j < len(b); j++ {
			if _, found := m[b[j]]; !found {
				set = append(set, b[j])
			}
		}
	}
	return set
}

// IsUint64Sorted verifies if a uint64 slice is sorted in ascending order.
func IsUint64Sorted(a []uint64) bool {
	if len(a) == 0 || len(a) == 1 {
		return true
	}
	for i := 1; i < len(a); i++ {
		if a[i-1] > a[i] {
			return false
		}
	}
	return true
}

// NotUint64 returns the uint64 in slice a that are
// not in slice b with time complexity of approximately
// O(n) leveraging a map to check for element existence
// off by a constant factor of underlying map efficiency.
func NotUint64(a []uint64, b []uint64) []uint64 {
	set := make([]uint64, 0)
	m := make(map[uint64]bool)

	for i := 0; i < len(a); i++ {
		m[a[i]] = true
	}
	for i := 0; i < len(b); i++ {
		if _, found := m[b[i]]; !found {
			set = append(set, b[i])
		}
	}
	return set
}

// IsInUint64 returns true if a is in b and False otherwise.
func IsInUint64(a uint64, b []uint64) bool {
	for _, v := range b {
		if a == v {
			return true
		}
	}
	return false
}

// IntersectionInt64 of any number of int64 slices with time
// complexity of approximately O(n) leveraging a map to
// check for element existence off by a constant factor
// of underlying map efficiency.
func IntersectionInt64(s ...[]int64) []int64 {
	if len(s) == 0 {
		return []int64{}
	}
	if len(s) == 1 {
		return s[0]
	}
	set := make([]int64, 0)
	m := make(map[int64]bool)
	for i := 1; i < len(s); i++ {
		for j := 0; j < len(s[i-1]); j++ {
			m[s[i-1][j]] = true
		}
		for j := 0; j < len(s[i]); j++ {
			if _, found := m[s[i][j]]; found {
				set = append(set, s[i][j])
			}
		}
	}
	return set
}

// UnionInt64 of any number of int64 slices with time
// complexity of approximately O(n) leveraging a map to
// check for element existence off by a constant factor
// of underlying map efficiency.
func UnionInt64(s ...[]int64) []int64 {
	if len(s) == 0 {
		return []int64{}
	}
	if len(s) == 1 {
		return s[0]
	}
	set := s[0]
	m := make(map[int64]bool)
	for i := 1; i < len(s); i++ {
		a := s[i-1]
		b := s[i]
		for j := 0; j < len(a); j++ {
			m[a[j]] = true
		}
		for j := 0; j < len(b); j++ {
			if _, found := m[b[j]]; !found {
				set = append(set, b[j])
			}
		}
	}
	return set
}

// NotInt64 returns the int64 in slice a that are
// not in slice b with time complexity of approximately
// O(n) leveraging a map to check for element existence
// off by a constant factor of underlying map efficiency.
func NotInt64(a []int64, b []int64) []int64 {
	set := make([]int64, 0)
	m := make(map[int64]bool)

	for i := 0; i < len(a); i++ {
		m[a[i]] = true
	}
	for i := 0; i < len(b); i++ {
		if _, found := m[b[i]]; !found {
			set = append(set, b[i])
		}
	}
	return set
}

// IsInInt64 returns true if a is in b and False otherwise.
func IsInInt64(a int64, b []int64) bool {
	for _, v := range b {
		if a == v {
			return true
		}
	}
	return false
}

// UnionByteSlices returns the common elements between sets of byte slices.
func UnionByteSlices(s ...[][]byte) [][]byte {
	if len(s) == 0 {
		return [][]byte{}
	}
	if len(s) == 1 {
		return s[0]
	}
	set := s[0]
	m := make(map[string]bool)
	for i := 1; i < len(s); i++ {
		for j := 0; j < len(s[i-1]); j++ {
			m[string(s[i-1][j])] = true
		}
		for j := 0; j < len(s[i]); j++ {
			if _, found := m[string(s[i][j])]; !found {
				set = append(set, s[i][j])
			}
		}
	}
	return set
}

// IntersectionByteSlices returns the common elements between sets of byte slices.
func IntersectionByteSlices(s ...[][]byte) [][]byte {
	if len(s) == 0 {
		return [][]byte{}
	}
	if len(s) == 1 {
		return s[0]
	}
	inter := make([][]byte, 0)
	for i := 1; i < len(s); i++ {
		hash := make(map[string]bool)
		for _, e := range s[i-1] {
			hash[string(e)] = true
		}
		for _, e := range s[i] {
			if hash[string(e)] {
				inter = append(inter, e)
			}
		}
		tmp := make([][]byte, 0)
		// Remove duplicates from slice.
		encountered := make(map[string]bool)
		for _, element := range inter {
			if !encountered[string(element)] {
				tmp = append(tmp, element)
				encountered[string(element)] = true
			}
		}
		inter = tmp
	}
	return inter
}

// SplitCommaSeparated values from the list. Example: []string{"a,b", "c,d"} becomes []string{"a", "b", "c", "d"}.
func SplitCommaSeparated(arr []string) []string {
	var result []string
	for _, val := range arr {
		result = append(result, strings.Split(val, ",")...)
	}
	return result
}
