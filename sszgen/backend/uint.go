package backend

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/sszgen/types"
)

type generateUint struct {
	valRep *types.ValueUint
	targetPackage string
	casterConfig
}

func (g *generateUint) coerce() func(string) string {
	return func(fieldName string) string {
		return fmt.Sprintf("%s(%s)", g.valRep.TypeName(), fieldName)
	}
}

func (g *generateUint) generateUnmarshalValue(fieldName string, offset string) string {
	// mispelling of Unmarshall due to misspelling of method exported by fastssz
	convert := fmt.Sprintf("ssz.UnmarshallUint%d(%s)", g.valRep.Size, offset)
	return fmt.Sprintf("%s = %s", fieldName, g.casterConfig.toOverlay(convert))
}

func (g *generateUint) generateFixedMarshalValue(fieldName string) string {
	return fmt.Sprintf("dst = ssz.MarshalUint%d(dst, %s)", g.valRep.Size, fieldName)
}

func (g *generateUint) generateHTRPutter(fieldName string) string {
	return fmt.Sprintf("hh.PutUint%d(%s)", g.valRep.Size, fieldName)
}

func (g *generateUint) variableSizeSSZ(fieldname string) string {
	return ""
}

var _ valueGenerator = &generateUint{}
