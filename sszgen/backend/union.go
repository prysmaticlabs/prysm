package backend

import (
	"github.com/prysmaticlabs/prysm/sszgen/types"
)

type generateUnion struct {
	*types.ValueUnion
	targetPackage string
}

func (g *generateUnion) generateHTRPutter(fieldName string) string {
	return ""
}

func (g *generateUnion) generateUnmarshalValue(fieldName string, s string) string {
	return ""
}

func (g *generateUnion) generateFixedMarshalValue(fieldName string) string {
	return ""
}

func (g *generateUnion) variableSizeSSZ(fieldname string) string {
	return ""
}

var _ valueGenerator = &generateUnion{}
