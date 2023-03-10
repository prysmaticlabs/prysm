package slice

import "strings"

// UnionByteSlices returns the all elements between sets of byte slices.
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
	m := make(map[string]int)
	for _, k := range s[0] {
		m[string(k)] = 1
	}
	for i, num := 1, len(s); i < num; i++ {
		for _, k := range s[i] {
			if _, found := m[string(k)]; found && i == m[string(k)] {
				m[string(k)]++
				if m[string(k)] == num {
					inter = append(inter, k)
				}
			}
		}
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
