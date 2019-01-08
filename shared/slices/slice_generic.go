package slices

import (
	"fmt"
	"reflect"
)

func interfaceToSlice(slice interface{}) []interface{} {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		panic("InterfaceSlice() given a non-slice type")
	}
	ret := make([]interface{}, s.Len())
	for i := 0; i < s.Len(); i++ {
		ret[i] = s.Index(i).Interface()
	}
	return ret
}

// GenericIntersection returns a new set with elements that are common in
// both sets a and b.
func GenericIntersection(a, b interface{}) (reflect.Value, error) {

	switch v := a.(type) {
	case []uint32:

		set := make([]uint32, 0)
		sliceType := reflect.TypeOf(set)
		modifiedSet := reflect.MakeSlice(sliceType, 0, 0)
		m := make(map[uint32]bool)

		set1 := interfaceToSlice(a)
		set2 := interfaceToSlice(b)
		for i := 0; i < len(set1); i++ {
			m[set1[i].(uint32)] = true
		}
		for i := 0; i < len(set2); i++ {
			if _, found := m[set2[i].(uint32)]; found {
				rv := reflect.ValueOf(set2[i])
				modifiedSet = reflect.Append(modifiedSet, rv)
			}
		}
		return modifiedSet, nil

	case []float32:

		set := make([]float32, 0)
		sliceType := reflect.TypeOf(set)
		modifiedSet := reflect.MakeSlice(sliceType, 0, 0)
		m := make(map[float32]bool)

		set1 := interfaceToSlice(a)
		set2 := interfaceToSlice(b)
		for i := 0; i < len(set1); i++ {
			m[set1[i].(float32)] = true
		}
		for i := 0; i < len(set2); i++ {
			if _, found := m[set2[i].(float32)]; found {
				rv := reflect.ValueOf(set2[i])
				modifiedSet = reflect.Append(modifiedSet, rv)
			}
		}
		return modifiedSet, nil

	case []int32:

		set := make([]int32, 0)
		sliceType := reflect.TypeOf(set)
		modifiedSet := reflect.MakeSlice(sliceType, 0, 0)
		m := make(map[int32]bool)

		set1 := interfaceToSlice(a)
		set2 := interfaceToSlice(b)
		for i := 0; i < len(set1); i++ {
			m[set1[i].(int32)] = true
		}
		for i := 0; i < len(set2); i++ {
			if _, found := m[set2[i].(int32)]; found {
				rv := reflect.ValueOf(set2[i])
				modifiedSet = reflect.Append(modifiedSet, rv)
			}
		}
		return modifiedSet, nil
	case []string:

		set := make([]string, 0)
		sliceType := reflect.TypeOf(set)
		modifiedSet := reflect.MakeSlice(sliceType, 0, 0)
		m := make(map[string]bool)

		set1 := interfaceToSlice(a)
		set2 := interfaceToSlice(b)
		for i := 0; i < len(set1); i++ {
			m[set1[i].(string)] = true
		}
		for i := 0; i < len(set2); i++ {
			if _, found := m[set2[i].(string)]; found {
				rv := reflect.ValueOf(set2[i])
				modifiedSet = reflect.Append(modifiedSet, rv)
			}
		}
		return modifiedSet, nil

	default:
		return reflect.ValueOf(interface{}(nil)), fmt.Errorf("slice error: for input type %v", v)
	}

}

// GenericUnion returns a new set with elements from both
// the given sets a and b.
func GenericUnion(a, b interface{}) (reflect.Value, error) {

	switch v := a.(type) {
	case []uint32:
		set := make([]uint32, 0)
		sliceType := reflect.TypeOf(set)
		modifiedSet := reflect.MakeSlice(sliceType, 0, 0)
		m := make(map[uint32]bool)

		set1 := interfaceToSlice(a)
		set2 := interfaceToSlice(b)
		for i := 0; i < len(set1); i++ {
			m[set1[i].(uint32)] = true
			rv := reflect.ValueOf(set1[i])
			modifiedSet = reflect.Append(modifiedSet, rv)
		}
		for i := 0; i < len(set2); i++ {
			if _, found := m[set2[i].(uint32)]; !found {
				rv := reflect.ValueOf(set2[i])
				modifiedSet = reflect.Append(modifiedSet, rv)
			}
		}
		return modifiedSet, nil
	case []float32:

		set := make([]float32, 0)
		sliceType := reflect.TypeOf(set)
		modifiedSet := reflect.MakeSlice(sliceType, 0, 0)
		m := make(map[float32]bool)

		set1 := interfaceToSlice(a)
		set2 := interfaceToSlice(b)
		for i := 0; i < len(set1); i++ {
			m[set1[i].(float32)] = true
			rv := reflect.ValueOf(set1[i])
			modifiedSet = reflect.Append(modifiedSet, rv)
		}
		for i := 0; i < len(set2); i++ {
			if _, found := m[set2[i].(float32)]; !found {
				rv := reflect.ValueOf(set2[i])
				modifiedSet = reflect.Append(modifiedSet, rv)
			}
		}
		return modifiedSet, nil
	case []int32:

		set := make([]int32, 0)
		sliceType := reflect.TypeOf(set)
		modifiedSet := reflect.MakeSlice(sliceType, 0, 0)
		m := make(map[int32]bool)

		set1 := interfaceToSlice(a)
		set2 := interfaceToSlice(b)
		for i := 0; i < len(set1); i++ {
			m[set1[i].(int32)] = true
			rv := reflect.ValueOf(set1[i])
			modifiedSet = reflect.Append(modifiedSet, rv)
		}
		for i := 0; i < len(set2); i++ {
			if _, found := m[set2[i].(int32)]; !found {
				rv := reflect.ValueOf(set2[i])
				modifiedSet = reflect.Append(modifiedSet, rv)
			}
		}
		return modifiedSet, nil

	case []string:

		set := make([]string, 0)
		sliceType := reflect.TypeOf(set)
		modifiedSet := reflect.MakeSlice(sliceType, 0, 0)
		m := make(map[string]bool)

		set1 := interfaceToSlice(a)
		set2 := interfaceToSlice(b)
		for i := 0; i < len(set1); i++ {
			m[set1[i].(string)] = true
			rv := reflect.ValueOf(set1[i])
			modifiedSet = reflect.Append(modifiedSet, rv)
		}
		for i := 0; i < len(set2); i++ {
			if _, found := m[set2[i].(string)]; !found {
				rv := reflect.ValueOf(set2[i])
				modifiedSet = reflect.Append(modifiedSet, rv)
			}
		}
		return modifiedSet, nil

	default:
		return reflect.ValueOf(interface{}(nil)), fmt.Errorf("slice error: for input type %v", v)
	}

}

// GenericNot returns new set with elements which of a which are not in
// set b.
func GenericNot(a, b interface{}) (reflect.Value, error) {
	switch v := a.(type) {
	case []uint32:

		set := make([]uint32, 0)
		sliceType := reflect.TypeOf(set)
		modifiedSet := reflect.MakeSlice(sliceType, 0, 0)
		m := make(map[uint32]bool)

		set1 := interfaceToSlice(a)
		set2 := interfaceToSlice(b)
		for i := 0; i < len(set1); i++ {
			m[set1[i].(uint32)] = true

		}
		for i := 0; i < len(set2); i++ {
			if _, found := m[set2[i].(uint32)]; !found {
				rv := reflect.ValueOf(set2[i])
				modifiedSet = reflect.Append(modifiedSet, rv)
			}
		}
		return modifiedSet, nil
	case []float32:

		set := make([]float32, 0)
		sliceType := reflect.TypeOf(set)
		modifiedSet := reflect.MakeSlice(sliceType, 0, 0)
		m := make(map[float32]bool)

		set1 := interfaceToSlice(a)
		set2 := interfaceToSlice(b)
		for i := 0; i < len(set1); i++ {
			m[set1[i].(float32)] = true

		}
		for i := 0; i < len(set2); i++ {
			if _, found := m[set2[i].(float32)]; !found {
				rv := reflect.ValueOf(set2[i])
				modifiedSet = reflect.Append(modifiedSet, rv)
			}
		}
		return modifiedSet, nil

	case []int32:

		set := make([]int32, 0)
		sliceType := reflect.TypeOf(set)
		modifiedSet := reflect.MakeSlice(sliceType, 0, 0)
		m := make(map[int32]bool)

		set1 := interfaceToSlice(a)
		set2 := interfaceToSlice(b)
		for i := 0; i < len(set1); i++ {
			m[set1[i].(int32)] = true

		}
		for i := 0; i < len(set2); i++ {
			if _, found := m[set2[i].(int32)]; !found {
				rv := reflect.ValueOf(set2[i])
				modifiedSet = reflect.Append(modifiedSet, rv)
			}
		}
		return modifiedSet, nil
	case []string:

		set := make([]string, 0)
		sliceType := reflect.TypeOf(set)
		modifiedSet := reflect.MakeSlice(sliceType, 0, 0)
		m := make(map[string]bool)

		set1 := interfaceToSlice(a)
		set2 := interfaceToSlice(b)
		for i := 0; i < len(set1); i++ {
			m[set1[i].(string)] = true

		}
		for i := 0; i < len(set2); i++ {
			if _, found := m[set2[i].(string)]; !found {
				rv := reflect.ValueOf(set2[i])
				modifiedSet = reflect.Append(modifiedSet, rv)
			}
		}
		return modifiedSet, nil

	default:
		return reflect.ValueOf(interface{}(nil)), fmt.Errorf("slice error: for input type %v", v)

	}

}

// GenericIsIn returns true if a is in b and False otherwise.
func GenericIsIn(a, b interface{}) bool {
	set1 := interfaceToSlice(b)
	for _, v := range set1 {
		if a == v {
			return true
		}
	}
	return false
}
