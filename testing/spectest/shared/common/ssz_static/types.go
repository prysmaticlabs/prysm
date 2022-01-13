package ssz_static

import (
	"testing"
)

type HTR func(interface{}) ([32]byte, error)

// SSZRoots is the format used to read spectest test data.
type SSZRoots struct {
	Root        string `json:"root"`
	SigningRoot string `json:"signing_root"`
}

// Unmarshaller determines the correct type per ObjectName and then hydrates the object from the
// serializedBytes. This method may call t.Skip if the type is not supported.
type Unmarshaller func(t *testing.T, serializedBytes []byte, objectName string) (interface{}, error)

type CustomHTRAdder func(t *testing.T, htrs []HTR, object interface{}) []HTR
