package slice

import "golang.org/x/exp/constraints"

// Subset[T comparable] returns true if the first array is
// completely contained in the second array with time
// complexity of approximately o(n).
func Subset[T comparable](a, b []T) bool {
	if len(a) > len(b) {
		return false
	}

	set := make(map[T]uint64, len(b))
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

// Intersection[T comparable] of any number of T slices with time
// complexity of approximately O(n) leveraging a map to
// check for element existence off by a constant factor
// of underlying map efficiency.
func Intersection[T comparable](s ...[]T) []T {
	if len(s) == 0 {
		return []T{}
	}
	if len(s) == 1 {
		return s[0]
	}
	intersect := make([]T, 0)
	m := make(map[T]int)
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

// Union[T comparable] of any number of T slices with time
// complexity of approximately O(n) leveraging a map to
// check for element existence off by a constant factor
// of underlying map efficiency.
func Union[T comparable](s ...[]T) []T {
	if len(s) == 0 {
		return []T{}
	}
	if len(s) == 1 {
		return s[0]
	}
	set := s[0]
	m := make(map[T]bool)
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

// Set[T comparable] returns a slice with only unique
// values from the provided list of indices.
func Set[T comparable](a []T) []T {
	// Remove duplicates indices.
	intMap := map[T]bool{}
	cleanedIndices := make([]T, 0, len(a))
	for _, idx := range a {
		if intMap[idx] {
			continue
		}
		intMap[idx] = true
		cleanedIndices = append(cleanedIndices, idx)
	}
	return cleanedIndices
}

// IsSorted[T comparable] verifies if a T slice is sorted in ascending order.
func IsSorted[T constraints.Ordered](a []T) bool {
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

// Not[T comparable] returns the T in slice b that are
// not in slice a with time complexity of approximately
// O(n) leveraging a map to check for element existence
// off by a constant factor of underlying map efficiency.
func Not[T comparable](a, b []T) []T {
	set := make([]T, 0)
	m := make(map[T]bool)

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

// IsIn[T comparable] returns true if a is in b and False otherwise.
func IsIn[T comparable](a T, b []T) bool {
	for _, v := range b {
		if a == v {
			return true
		}
	}
	return false
}

// Unique[T comparable] returns an array with duplicates filtered based on the type given
func Unique[T comparable](a []T) []T {
	if a == nil || len(a) <= 1 {
		return a
	}
	found := map[T]bool{}
	result := make([]T, len(a))
	end := 0
	for i := 0; i < len(a); i++ {
		if !found[a[i]] {
			found[a[i]] = true
			result[end] = a[i]
			end += 1
		}
	}
	return result[:end]
}

// Reverse reverses any slice in place
// Taken from https://github.com/faiface/generics/blob/8cf65f0b43803410724d8c671cb4d328543ba07d/examples/sliceutils/sliceutils.go
func Reverse[E any](s []E) []E {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}
