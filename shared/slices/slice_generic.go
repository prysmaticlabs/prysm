package slices

// Type is the placeholder type that indicates a generic value.
// when executed variables of this type will be replaced with,
// references to the specific types.
type Type interface{}

// GenericItem is of type empty interface in order
// to hold any values.
type GenericItem Type

// GenericIntersection returns a new set with elements that are common in
// both sets a and b.
func GenericIntersection(a, b []GenericItem) []GenericItem {
	set := make([]GenericItem, 0)
	m := make(map[GenericItem]bool)

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

// GenericUnion returns a new set with elements from both
// the given sets a and b.
func GenericUnion(a, b []GenericItem) []GenericItem {
	set := make([]GenericItem, 0)
	m := make(map[GenericItem]bool)

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

// GenericNot returns new set with elements which of a which are not in
// set b.
func GenericNot(a, b []GenericItem) []GenericItem {
	set := make([]GenericItem, 0)
	m := make(map[GenericItem]bool)

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

// GenericIsIn returns true if a is in b and False otherwise.
func GenericIsIn(a GenericItem, b []GenericItem) bool {
	for _, v := range b {
		if a == v {
			return true
		}
	}
	return false
}
