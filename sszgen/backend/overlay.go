package backend

import (
	"fmt"
	"github.com/prysmaticlabs/prysm/sszgen/types"
)

type generateOverlay struct {
	*types.ValueOverlay
	targetPackage string
}

func (g *generateOverlay) toOverlay() func(string) string {
	wrapper := g.TypeName()
	if g.targetPackage != g.PackagePath() {
		wrapper = importAlias(g.PackagePath()) + "." + wrapper
	}
	return func(value string) string {
		return fmt.Sprintf("%s(%s)", wrapper, value)
	}
}

func (g *generateOverlay) generateVariableMarshalValue(fieldName string) string {
	gg := newValueGenerator(g.Underlying, g.targetPackage)
	vm, ok := gg.(variableMarshaller)
	if !ok {
		return ""
	}
	return vm.generateVariableMarshalValue(fieldName)
}

func (g *generateOverlay) generateUnmarshalValue(fieldName string, sliceName string) string {
	gg := newValueGenerator(g.Underlying, g.targetPackage)
	c, ok := gg.(caster)
	if ok {
		c.setToOverlay(g.toOverlay())
	}
	umv := gg.generateUnmarshalValue(fieldName, sliceName)
	if g.IsBitfield() {
		switch t := g.Underlying.(type) {
		case *types.ValueList:
			return fmt.Sprintf(`if err = ssz.ValidateBitlist(%s, %d); err != nil {
return err
}
%s`, sliceName, t.MaxSize, umv)
		}
	}
	return umv
}

func (g *generateOverlay) generateFixedMarshalValue(fieldName string) string {
	gg := newValueGenerator(g.Underlying, g.targetPackage)
	uc, ok := gg.(coercer)
	if ok {
		return gg.generateFixedMarshalValue(uc.coerce()(fieldName))
	}
	return gg.generateFixedMarshalValue(fieldName)
}

func (g *generateOverlay) variableSizeSSZ(fieldname string) string {
	return ""
}

func (g *generateOverlay) generateHTRPutter(fieldName string) string {
	if g.IsBitfield() && g.Name == "Bitlist" {
		ul, ok := g.Underlying.(*types.ValueList)
		if !ok {
			panic(fmt.Sprintf("unexpected underlying type for Bitlist, expected ValueList, got %v", g.Underlying))
		}
		t := `if len(%s) == 0 {
	return ssz.ErrEmptyBitlist
}
hh.PutBitlist(%s, %d)`
		return fmt.Sprintf(t, fieldName, fieldName, ul.MaxSize)
	}
	gg := newValueGenerator(g.Underlying, g.targetPackage)
	htrp, ok := gg.(htrPutter)
	if !ok {
		return ""
	}
	uc, ok := gg.(coercer)
	if ok {
		c := uc.coerce()
		return htrp.generateHTRPutter(c(fieldName))
	}
	return htrp.generateHTRPutter(fieldName)
}

var _ valueGenerator = &generateOverlay{}
