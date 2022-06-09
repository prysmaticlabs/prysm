package backend

import (
	"fmt"
	"github.com/prysmaticlabs/prysm/sszgen/types"
)

type generateBool struct {
	valRep *types.ValueBool
	targetPackage string
	casterConfig
}

func (g *generateBool) generateHTRPutter(fieldName string) string {
	return fmt.Sprintf("hh.PutBool(%s)", fieldName)
}

func (g *generateBool) coerce() func(string) string {
	return func(fieldName string) string {
		return fmt.Sprintf("%s(%s)", g.valRep.TypeName(), fieldName)
	}
}

func (g *generateBool) generateFixedMarshalValue(fieldName string) string {
	return fmt.Sprintf("dst = ssz.MarshalBool(dst, %s)", fieldName)
}

func (g *generateBool) generateUnmarshalValue(fieldName string, offset string) string {
	convert := fmt.Sprintf("ssz.UnmarshalBool(%s)", offset)
	return fmt.Sprintf("%s = %s", fieldName, g.casterConfig.toOverlay(convert))
}

func (g *generateBool) variableSizeSSZ(fieldname string) string {
	return ""
}

var _ valueGenerator = &generateBool{}
