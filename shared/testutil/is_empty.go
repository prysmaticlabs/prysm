package testutil

import "reflect"

// IsEmpty returns true if the struct is empty.
func IsEmpty(item interface{}) bool {
	val := reflect.ValueOf(item)
	for i := 0; i < val.NumField(); i++ {
		if !reflect.DeepEqual(val.Field(i).Interface(), reflect.Zero(val.Field(i).Type()).Interface()) {
			return false
		}
	}
	return true
}
