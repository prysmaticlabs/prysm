package slice

// SubsetUint64_old returns true if the first array is
// completely contained in the second array with time
// complexity of approximately o(n).
func SubsetUint64_old(a, b []uint64) bool {
	if len(a) > len(b) {
		return false
	}

	set := make(map[uint64]uint64, len(b))
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

// IntersectionUint64_old of any number of uint64 slices with time
// complexity of approximately O(n) leveraging a map to
// check for element existence off by a constant factor
// of underlying map efficiency.
func IntersectionUint64_old(s ...[]uint64) []uint64 {
	if len(s) == 0 {
		return []uint64{}
	}
	if len(s) == 1 {
		return s[0]
	}
	intersect := make([]uint64, 0)
	m := make(map[uint64]int)
	for _, k := range s[0] {
		m[k] = 1
	}
	for i, num := 1, len(s); i < num; i++ {
		for _, k := range s[i] {
			// Increment and check only if item is present in both, and no increment has happened yet.
			if _, found := m[k]; found && i == m[k] {
				m[k]++
				if m[k] == num {
					intersect = append(intersect, k)
				}
			}
		}
	}
	return intersect
}

// UnionUint64_old of any number of uint64 slices with time
// complexity of approximately O(n) leveraging a map to
// check for element existence off by a constant factor
// of underlying map efficiency.
func UnionUint64_old(s ...[]uint64) []uint64 {
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

// SetUint64_old returns a slice with only unique
// values from the provided list of indices.
func SetUint64_old(a []uint64) []uint64 {
	// Remove duplicates indices.
	intMap := map[uint64]bool{}
	cleanedIndices := make([]uint64, 0, len(a))
	for _, idx := range a {
		if intMap[idx] {
			continue
		}
		intMap[idx] = true
		cleanedIndices = append(cleanedIndices, idx)
	}
	return cleanedIndices
}

// IsUint64_oldSorted verifies if a uint64 slice is sorted in ascending order.
func IsUint64_oldSorted(a []uint64) bool {
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

// NotUint64_old returns the uint64 in slice b that are
// not in slice a with time complexity of approximately
// O(n) leveraging a map to check for element existence
// off by a constant factor of underlying map efficiency.
func NotUint64_old(a, b []uint64) []uint64 {
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

// IsInUint64_old returns true if a is in b and False otherwise.
func IsInUint64_old(a uint64, b []uint64) bool {
	for _, v := range b {
		if a == v {
			return true
		}
	}
	return false
}
