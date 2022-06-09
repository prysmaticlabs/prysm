package backend

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/prysmaticlabs/prysm/sszgen/types"
)

type generateVector struct {
	valRep *types.ValueVector
	targetPackage string
	casterConfig
}

func (g *generateVector) generateUnmarshalValue(fieldName string, sliceName string) string {
	gg := newValueGenerator(g.valRep.ElementValue, g.targetPackage)
	switch g.valRep.ElementValue.(type) {
	case *types.ValueByte:
		t := `%s = make([]byte, 0, %d)
%s = append(%s, %s...)`
		return fmt.Sprintf(t, fieldName, g.valRep.Size, fieldName, fieldName, g.casterConfig.toOverlay(sliceName))
	default:
		loopVar := "i"
		if fieldName[0:1] == "i" && monoCharacter(fieldName) {
			loopVar = fieldName + "i"
		}
		t := `{
	var tmp {{ .TypeName }}
	{{.FieldName}} = make([]{{.TypeName}}, {{.NumElements}})
	for {{ .LoopVar }} := 0; {{ .LoopVar }} < {{ .NumElements }}; {{ .LoopVar }} ++ {
		tmpSlice := {{ .SliceName }}[{{ .LoopVar }}*{{ .NestedFixedSize }}:(1+{{ .LoopVar }})*{{ .NestedFixedSize }}]
{{ .NestedUnmarshal }}
		{{ .FieldName }}[{{.LoopVar}}] = tmp
	}
}`
		tmpl, err := template.New("tmplgenerateUnmarshalValueDefault").Parse(t)
		if err != nil {
			panic(err)
		}
		buf := bytes.NewBuffer(nil)
		nvr := g.valRep.ElementValue
		err = tmpl.Execute(buf, struct{
			TypeName string
			SliceName string
			NumElements int
			NestedFixedSize int
			LoopVar string
			NestedUnmarshal string
			FieldName string
		}{
			TypeName: fullyQualifiedTypeName(nvr, g.targetPackage),
			SliceName: sliceName,
			NumElements: g.valRep.FixedSize() / g.valRep.ElementValue.FixedSize(),
			NestedFixedSize: g.valRep.ElementValue.FixedSize(),
			LoopVar: loopVar,
			NestedUnmarshal: gg.generateUnmarshalValue("tmp", "tmpSlice"),
			FieldName: fieldName,
		})
		if err != nil {
			panic(err)
		}
		return string(buf.Bytes())
	}
}

var tmplGenerateMarshalValueVector = `if len({{.FieldName}}) != {{.Size}} {
	return nil, ssz.ErrBytesLength
}
{{.MarshalValue}}`

func (g *generateVector) generateFixedMarshalValue(fieldName string) string {
	mvTmpl, err := template.New("tmplGenerateMarshalValueVector").Parse(tmplGenerateMarshalValueVector)
	if err != nil {
		panic(err)
	}
	var marshalValue string
	switch g.valRep.ElementValue.(type) {
	case *types.ValueByte:
		marshalValue = fmt.Sprintf("dst = append(dst, %s...)", fieldName)
	default:
		nestedFieldName := "o"
		if fieldName[0:1] == "o" && monoCharacter(fieldName) {
			nestedFieldName = fieldName + "o"
		}
		t := `for _, %s := range %s {
	%s
}`
		gg := newValueGenerator(g.valRep.ElementValue, g.targetPackage)
		internal := gg.generateFixedMarshalValue(nestedFieldName)
		marshalValue = fmt.Sprintf(t, nestedFieldName, fieldName, internal)
	}
	buf := bytes.NewBuffer(nil)
	err = mvTmpl.Execute(buf, struct{
		FieldName string
		Size int
		MarshalValue string
	}{
		FieldName: fieldName,
		Size: g.valRep.Size,
		MarshalValue: marshalValue,
	})
	if err != nil {
		panic(err)
	}
	return string(buf.Bytes())
}

var generateVectorHTRPutterTmpl = `{
	if len({{.FieldName}}) != {{.Size}} {
		return ssz.ErrVectorLength
	}
	subIndx := hh.Index()
	for _, {{.NestedFieldName}} := range {{.FieldName}} {
		{{.AppendCall}}
	}
	{{.Merkleize}}
}`

type vecPutterElements struct {
	FieldName string
	NestedFieldName string
	Size int
	AppendCall string
	Merkleize string
}

func renderHtrVecPutter(lpe vecPutterElements) string {
	tmpl, err := template.New("renderHtrVecPutter").Parse(generateVectorHTRPutterTmpl)
	if err != nil {
		panic(err)
	}
	buf := bytes.NewBuffer(nil)
	err = tmpl.Execute(buf, lpe)
	if err != nil {
		panic(err)
	}
	return buf.String()
}

func (g *generateVector) isByteVector() bool {
	_, isByte := g.valRep.ElementValue.(*types.ValueByte)
	return isByte
}

func (g *generateVector) renderByteSliceAppend(fieldName string) string {
	t := `if len(%s) != %d {
	return ssz.ErrBytesLength
}
hh.Append(%s)`
	return fmt.Sprintf(t, fieldName, g.valRep.Size, fieldName)
}

func (g *generateVector) generateHTRPutter(fieldName string) string {
	nestedFieldName := "o"
	if fieldName[0:1] == "o" && monoCharacter(fieldName) {
		nestedFieldName = fieldName + "o"
	}

	// resolve pointers and overlays to their underlying types
	vr := g.valRep.ElementValue
	if vrp, isPointer := vr.(*types.ValuePointer); isPointer {
		vr = vrp.Referent
	}
	if vro, isOverlay := vr.(*types.ValueOverlay); isOverlay {
		vr = vro.Underlying
	}

	vpe := vecPutterElements{
		FieldName: fieldName,
		NestedFieldName: nestedFieldName,
		Size: g.valRep.Size,
	}

	switch v := vr.(type) {
	case *types.ValueByte:
		t := `if len(%s) != %d {
			return ssz.ErrBytesLength
		}
		hh.PutBytes(%s)`
		return fmt.Sprintf(t, fieldName, g.valRep.Size, fieldName)
	case *types.ValueVector:
		gv := &generateVector{valRep: v, targetPackage: g.targetPackage}
		if gv.isByteVector() {
			vpe.AppendCall = gv.renderByteSliceAppend(nestedFieldName)
			vpe.Merkleize = "hh.Merkleize(subIndx)"
			return renderHtrVecPutter(vpe)
		}
	case *types.ValueUint:
		vpe.AppendCall = fmt.Sprintf("hh.AppendUint%d(%s)", v.Size, nestedFieldName)
		vpe.Merkleize = "hh.Merkleize(subIndx)"
		return renderHtrVecPutter(vpe)
	default:
		panic(fmt.Sprintf("unsupported type combination - vector of %v", v))
	}
	return ""
}

func monoCharacter(s string) bool {
	ch := s[0]
	for i := 1; i < len(s); i++ {
		if s[i] == ch {
			continue
		}
		return false
	}
	return true
}

func (g *generateVector) variableSizeSSZ(fieldName string) string {
	if !g.valRep.ElementValue.IsVariableSized() {
		return fmt.Sprintf("len(%s) * %d", fieldName, g.valRep.ElementValue.FixedSize())
	}
	return ""
}

func (g *generateVector) coerce() func(string) string {
	return func(fieldName string) string {
		return fmt.Sprintf("%s(%s)", g.valRep.TypeName(), fieldName)
	}
}

var _ valueGenerator = &generateVector{}