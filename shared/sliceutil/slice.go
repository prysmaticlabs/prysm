package sliceutil

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

// IntersectionUint64 of two uint64 slices with time
// complexity of approximately O(n) leveraging a map to
// check for element existence off by a constant factor
// of underlying map efficiency.
func IntersectionUint64(a []uint64, b []uint64) []uint64 {
	set := make([]uint64, 0)
	m := make(map[uint64]bool)

	for i := 0; i < len(a); i++ {
		m[a[i]] = true
	}
	for i := 0; i < len(b); i++ {
		if _, found := m[b[i]]; found {
			set = append(set, b[i])
		}
	}
	return set
}

// UnionUint64 of two uint64 slices with time
// complexity of approximately O(n) leveraging a map to
// check for element existence off by a constant factor
// of underlying map efficiency.
func UnionUint64(a []uint64, b []uint64) []uint64 {
	set := make([]uint64, 0)
	m := make(map[uint64]bool)

	for i := 0; i < len(a); i++ {
		m[a[i]] = true
		set = append(set, a[i])
	}
	for i := 0; i < len(b); i++ {
		if _, found := m[b[i]]; !found {
			set = append(set, b[i])
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

// IntersectionInt64 of two int64 slices with time
// complexity of approximately O(n) leveraging a map to
// check for element existence off by a constant factor
// of underlying map efficiency.
func IntersectionInt64(a []int64, b []int64) []int64 {
	set := make([]int64, 0)
	m := make(map[int64]bool)

	for i := 0; i < len(a); i++ {
		m[a[i]] = true
	}
	for i := 0; i < len(b); i++ {
		if _, found := m[b[i]]; found {
			set = append(set, b[i])
		}
	}
	return set
}

// UnionInt64 of two int64 slices with time
// complexity of approximately O(n) leveraging a map to
// check for element existence off by a constant factor
// of underlying map efficiency.
func UnionInt64(a []int64, b []int64) []int64 {
	set := make([]int64, 0)
	m := make(map[int64]bool)

	for i := 0; i < len(a); i++ {
		m[a[i]] = true
		set = append(set, a[i])
	}
	for i := 0; i < len(b); i++ {
		if _, found := m[b[i]]; !found {
			set = append(set, b[i])
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

// IntersectionByteSlices returns the common elements between two
// sets of byte slices.
func IntersectionByteSlices(s1, s2 [][]byte) [][]byte {
	hash := make(map[string]bool)
	for _, e := range s1 {
		hash[string(e)] = true
	}
	inter := make([][]byte, 0)
	for _, e := range s2 {
		if hash[string(e)] {
			inter = append(inter, e)
		}
	}
	// Remove duplicates from slice.
	deduped := make([][]byte, 0)
	encountered := make(map[string]bool)
	for _, element := range inter {
		if !encountered[string(element)] {
			deduped = append(deduped, element)
			encountered[string(element)] = true
		}
	}
	return deduped
}

// TotalIntersectionByteSlices takes in a set of byte slices
// and determines the intersection of common elements across all of them,
// returning a single slice of byte slices.
func TotalIntersectionByteSlices(sets [][][]byte) [][]byte {
	if len(sets) == 0 {
		return [][]byte{}
	}
	if len(sets) == 1 {
		return sets[0]
	}
	intersected := IntersectionByteSlices(sets[0], sets[1])
	for i := 2; i < len(sets); i++ {
		intersected = IntersectionByteSlices(intersected, sets[i])
	}
	return intersected
}
