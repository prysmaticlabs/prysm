package sliceutil

import (
	"fmt"
	"reflect"
)

func interfaceToSlice(slice interface{}) ([]interface{}, error) {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		return nil, fmt.Errorf("slice error: not of type slice")
	}
	ret := make([]interface{}, s.Len())
	for i := 0; i < s.Len(); i++ {
		ret[i] = s.Index(i).Interface()
	}
	return ret, nil
}

// GenericIntersection returns a new set with elements that are common in
// both sets a and b.
func GenericIntersection(a, b interface{}) (reflect.Value, error) {

	set := reflect.MakeSlice(reflect.TypeOf(a), 0, 0)
	set1, err1 := interfaceToSlice(a)
	set2, err2 := interfaceToSlice(b)

	if err1 != nil {
		return set, fmt.Errorf("slice type is invalid %v", err1)
	}

	if err2 != nil {
		return set, fmt.Errorf("slice type is invalid %v", err2)
	}
	if len(set1) == 0 || len(set2) == 0 {
		return set, nil
	}

	m := reflect.MapOf(reflect.TypeOf(set1[0]), reflect.TypeOf(true))
	m1 := reflect.MakeMapWithSize(m, 0)
	for i := 0; i < len(set1); i++ {
		m1.SetMapIndex(reflect.ValueOf(set1[i]), reflect.ValueOf(true))
	}

	for i := 0; i < len(set2); i++ {
		x := m1.MapIndex(reflect.ValueOf(set2[i]))
		if x.IsValid() {
			if found := x; found.Bool() {
				rv := reflect.ValueOf(set2[i])
				set = reflect.Append(set, rv)
			}
		}
	}

	return set, nil
}

// GenericUnion returns a new set with elements from both
// the given sets a and b.
func GenericUnion(a, b interface{}) (reflect.Value, error) {

	set := reflect.MakeSlice(reflect.TypeOf(a), 0, 0)
	set1, err1 := interfaceToSlice(a)
	set2, err2 := interfaceToSlice(b)

	if err1 != nil {
		return set, fmt.Errorf("slice type is invalid %v", err1)
	}

	if err2 != nil {
		return set, fmt.Errorf("slice type is invalid %v", err2)
	}

	if len(set1) == 0 {
		return reflect.ValueOf(set2), nil
	}
	if len(set2) == 0 {
		return reflect.ValueOf(set1), nil
	}

	m := reflect.MapOf(reflect.TypeOf(set1[0]), reflect.TypeOf(true))
	m1 := reflect.MakeMapWithSize(m, 0)
	for i := 0; i < len(set1); i++ {
		m1.SetMapIndex(reflect.ValueOf(set1[i]), reflect.ValueOf(true))
		rv := reflect.ValueOf(set1[i])
		set = reflect.Append(set, rv)
	}

	for i := 0; i < len(set2); i++ {
		x := m1.MapIndex(reflect.ValueOf(set2[i]))
		if x.IsValid() {
			if found := x; !found.Bool() {
				rv := reflect.ValueOf(set2[i])
				set = reflect.Append(set, rv)
			}
		}
	}

	return set, nil

}

// GenericNot returns new set with elements which of a which are not in
// set b.
func GenericNot(a, b interface{}) (reflect.Value, error) {
	set := reflect.MakeSlice(reflect.TypeOf(a), 0, 0)
	set1, err1 := interfaceToSlice(a)
	set2, err2 := interfaceToSlice(b)

	if err1 != nil {
		return set, fmt.Errorf("slice type is invalid %v", err1)
	}

	if err2 != nil {
		return set, fmt.Errorf("slice type is invalid %v", err2)
	}

	if len(set1) == 0 {
		return reflect.ValueOf(set2), nil
	}
	if len(set2) == 0 {
		return reflect.ValueOf(set1), nil
	}

	m := reflect.MapOf(reflect.TypeOf(set1[0]), reflect.TypeOf(true))
	m1 := reflect.MakeMapWithSize(m, 0)
	for i := 0; i < len(set1); i++ {
		m1.SetMapIndex(reflect.ValueOf(set1[i]), reflect.ValueOf(true))

	}

	for i := 0; i < len(set2); i++ {
		x := m1.MapIndex(reflect.ValueOf(set2[i]))
		if x.IsValid() {
			if found := x; !found.Bool() {
				rv := reflect.ValueOf(set2[i])
				set = reflect.Append(set, rv)
			}
		}
	}

	return set, nil

}

// GenericIsIn returns true if a is in b and False otherwise.
func GenericIsIn(a, b interface{}) bool {
	set1, err := interfaceToSlice(b)
	if err == nil {
		for _, v := range set1 {
			if a == v {
				return true
			}
		}
	}

	return false
}
