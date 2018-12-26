package slices

// Intersection of two uint32 slices with time
// complexity of approximately O(n) leveraging a map to
// check for element existence off by a constant factor
// of underlying map efficiency.
func Intersection(a []uint32, b []uint32) []uint32 {
	set := make([]uint32, 0)
	m := make(map[uint32]bool)

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

// Union of two uint32 slices with time
// complexity of approximately O(n) leveraging a map to
// check for element existence off by a constant factor
// of underlying map efficiency.
func Union(a []uint32, b []uint32) []uint32 {
	set := make([]uint32, 0)
	m := make(map[uint32]bool)

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
