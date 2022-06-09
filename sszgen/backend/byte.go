package backend

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/sszgen/types"
)

type generateByte struct {
	*types.ValueByte
	targetPackage string
}

func (g *generateByte) generateHTRPutter(fieldName string) string {
	return ""
}

func (g *generateByte) coerce() func(string) string {
	return func(fieldName string) string {
		return fmt.Sprintf("%s(%s)", g.TypeName(), fieldName)
	}
}

func (g *generateByte) generateFixedMarshalValue(fieldName string) string {
	return ""
}

func (g *generateByte) generateUnmarshalValue(fieldName string, s string) string {
	return ""
}

func (g *generateByte) variableSizeSSZ(fieldname string) string {
	return ""
}

var _ valueGenerator = &generateByte{}
