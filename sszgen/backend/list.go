package backend

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/prysmaticlabs/prysm/sszgen/types"
)

type generateList struct {
	valRep *types.ValueList
	targetPackage string
	casterConfig
}

var generateListHTRPutterTmpl = `{
	if len({{.FieldName}}) > {{.MaxSize}} {
		return ssz.ErrListTooBig
	}
	subIndx := hh.Index()
	for _, {{.NestedFieldName}} := range {{.FieldName}} {
		{{.AppendCall}}
	}
	{{- .PadCall}}
	{{.Merkleize}}
}`

type listPutterElements struct {
	FieldName string
	NestedFieldName string
	MaxSize int
	AppendCall string
	PadCall string
	Merkleize string
}

func renderHtrListPutter(lpe listPutterElements) string {
	tmpl, err := template.New("renderHtrListPutter").Parse(generateListHTRPutterTmpl)
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

func (g *generateList) generateHTRPutter(fieldName string) string {
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

	lpe := listPutterElements{
		FieldName: fieldName,
		NestedFieldName: nestedFieldName,
		MaxSize: g.valRep.MaxSize,
	}
	switch v := vr.(type) {
	case *types.ValueByte:
		t := `if len(%s) > %d {
			return ssz.ErrBytesLength
		}
		hh.PutBytes(%s)`
		return fmt.Sprintf(t, fieldName, g.valRep.MaxSize, fieldName)
	case *types.ValueVector:
		gv := &generateVector{valRep: v, targetPackage: g.targetPackage}
		if gv.isByteVector() {
			lpe.AppendCall = gv.renderByteSliceAppend(nestedFieldName)
			mtmpl := `numItems := uint64(len(%s))
		hh.MerkleizeWithMixin(subIndx, numItems, ssz.CalculateLimit(%d, numItems, %d))`
			lpe.Merkleize = fmt.Sprintf(mtmpl, fieldName, g.valRep.MaxSize, v.FixedSize())
			return renderHtrListPutter(lpe)
		}
	case *types.ValueUint:
		lpe.AppendCall = fmt.Sprintf("hh.AppendUint%d(%s)", v.Size, nestedFieldName)
		if v.FixedSize() % ChunkSize != 0 {
			lpe.PadCall = "\nhh.FillUpTo32()"
		}
		mtmpl := `numItems := uint64(len(%s))
		hh.MerkleizeWithMixin(subIndx, numItems, ssz.CalculateLimit(%d, numItems, %d))`
		lpe.Merkleize = fmt.Sprintf(mtmpl, fieldName, g.valRep.MaxSize, v.FixedSize())
		return renderHtrListPutter(lpe)
	case *types.ValueContainer:
		gc := newValueGenerator(v, g.targetPackage)
		lpe.AppendCall = gc.generateHTRPutter(nestedFieldName)
		lpe.Merkleize = fmt.Sprintf("hh.MerkleizeWithMixin(subIndx, uint64(len(%s)), %d)", fieldName, g.valRep.MaxSize)
		return renderHtrListPutter(lpe)
	default:
		panic(fmt.Sprintf("unsupported type combination - list of %v", v))
	}
	return ""
}

var generateListGenerateUnmarshalValueFixedTmpl = `{
	if len({{.SliceName}}) % {{.ElementSize}} != 0 {
		return fmt.Errorf("misaligned bytes: {{.FieldName}} length is %d, which is not a multiple of {{.ElementSize}}", len({{.SliceName}}))
	}
	numElem := len({{.SliceName}}) / {{.ElementSize}}
	if numElem > {{ .MaxSize }} {
		return fmt.Errorf("ssz-max exceeded: {{.FieldName}} has %d elements, ssz-max is {{.MaxSize}}", numElem)
	}
	{{.FieldName}} = make([]{{.TypeName}}, numElem)
	for {{.LoopVar}} := 0; {{.LoopVar}} < numElem; {{.LoopVar}}++ {
		var tmp {{.TypeName}}
		{{.Initializer}}
		tmpSlice := {{.SliceName}}[{{.LoopVar}}*{{.NestedFixedSize}}:(1+{{.LoopVar}})*{{.NestedFixedSize}}]
	{{.NestedUnmarshal}}
		{{.FieldName}}[{{.LoopVar}}] = tmp
	}
}`

var generateListGenerateUnmarshalValueVariableTmpl = `{
// empty lists are zero length, so make sure there is room for an offset
// before attempting to unmarshal it
if len({{.SliceName}}) > 3 {
	firstOffset := ssz.ReadOffset({{.SliceName}}[0:4])
	if firstOffset % 4 != 0 {
			return fmt.Errorf("misaligned list bytes: when decoding {{.FieldName}}, end-of-list offset is %d, which is not a multiple of 4 (offset size)", firstOffset)
	}
	listLen := firstOffset / 4
	if listLen > {{.MaxSize}} {
			return fmt.Errorf("ssz-max exceeded: {{.FieldName}} has %d elements, ssz-max is {{.MaxSize}}", listLen)
	}
	listOffsets := make([]uint64, listLen)
	for {{.LoopVar}} := 0; uint64({{.LoopVar}}) < listLen; {{.LoopVar}}++ {
		listOffsets[{{.LoopVar}}] = ssz.ReadOffset({{.SliceName}}[{{.LoopVar}}*4:({{.LoopVar}}+1)*4])
	}
	{{.FieldName}} = make([]{{.TypeName}}, len(listOffsets))
	for {{.LoopVar}} := 0; {{.LoopVar}} < len(listOffsets); {{.LoopVar}}++ {
			var tmp {{.TypeName}}
			{{.Initializer}}
			var tmpSlice []byte
			if {{.LoopVar}}+1 == len(listOffsets) {
				tmpSlice = {{.SliceName}}[listOffsets[{{.LoopVar}}]:]
			} else {
				tmpSlice = {{.SliceName}}[listOffsets[{{.LoopVar}}]:listOffsets[{{.LoopVar}}+1]]
			}
		{{.NestedUnmarshal}}
			{{.FieldName}}[{{.LoopVar}}] = tmp
	}
}
}`

func (g *generateList) generateUnmarshalVariableValue(fieldName string, sliceName string) string {
	loopVar := "i"
	if fieldName[0:1] == "i" && monoCharacter(fieldName) {
		loopVar = fieldName + "i"
	}
	gg := newValueGenerator(g.valRep.ElementValue, g.targetPackage)
	vi, ok := gg.(valueInitializer)
	var initializer string
	if ok {
		initializer = vi.initializeValue("tmp")
		if initializer != "" {
			initializer = "tmp = " + initializer
		}
	}
	tmpl, err := template.New("generateListGenerateUnmarshalValueVariableTmpl").Parse(generateListGenerateUnmarshalValueVariableTmpl)
	if err != nil {
		panic(err)
	}
	buf := bytes.NewBuffer(nil)
	err = tmpl.Execute(buf, struct{
		LoopVar string
		SliceName string
		ElementSize int
		TypeName string
		FieldName string
		MaxSize int
		Initializer string
		NestedFixedSize int
		NestedUnmarshal string
	}{
		LoopVar: loopVar,
		SliceName: sliceName,
		ElementSize: g.valRep.ElementValue.FixedSize(),
		TypeName: fullyQualifiedTypeName(g.valRep.ElementValue, g.targetPackage),
		FieldName: fieldName,
		MaxSize: g.valRep.MaxSize,
		Initializer: initializer,
		NestedFixedSize: g.valRep.ElementValue.FixedSize(),
		NestedUnmarshal: gg.generateUnmarshalValue("tmp", "tmpSlice"),
	})
	if err != nil {
		panic(err)
	}
	return buf.String()
}

func (g *generateList) generateUnmarshalFixedValue(fieldName string, sliceName string) string {
	loopVar := "i"
	if fieldName[0:1] == "i" && monoCharacter(fieldName) {
		loopVar = fieldName + "i"
	}
	gg := newValueGenerator(g.valRep.ElementValue, g.targetPackage)
	nestedUnmarshal := ""
	switch g.valRep.ElementValue.(type) {
	case *types.ValueByte:
		return fmt.Sprintf("%s = append([]byte{}, %s...)", fieldName, g.casterConfig.toOverlay(sliceName))
	default:
		nestedUnmarshal = gg.generateUnmarshalValue("tmp", "tmpSlice")
	}
	vi, ok := gg.(valueInitializer)
	var initializer string
	if ok {
		initializer = vi.initializeValue("tmp")
		if initializer != "" {
			initializer = "tmp = " + initializer
		}
	}
	tmpl, err := template.New("generateListGenerateUnmarshalValueFixedTmpl").Parse(generateListGenerateUnmarshalValueFixedTmpl)
	if err != nil {
		panic(err)
	}
	buf := bytes.NewBuffer(nil)
	err = tmpl.Execute(buf, struct{
		LoopVar string
		SliceName string
		ElementSize int
		TypeName string
		FieldName string
		MaxSize int
		Initializer string
		NestedFixedSize int
		NestedUnmarshal string
	}{
		LoopVar: loopVar,
		SliceName: sliceName,
		ElementSize: g.valRep.ElementValue.FixedSize(),
		TypeName: fullyQualifiedTypeName(g.valRep.ElementValue, g.targetPackage),
		FieldName: fieldName,
		MaxSize: g.valRep.MaxSize,
		Initializer: initializer,
		NestedFixedSize: g.valRep.ElementValue.FixedSize(),
		NestedUnmarshal: nestedUnmarshal,
	})
	if err != nil {
		panic(err)
	}
	return buf.String()
}

func (g *generateList) generateUnmarshalValue(fieldName string, sliceName string) string {
	if g.valRep.ElementValue.IsVariableSized() {
		return g.generateUnmarshalVariableValue(fieldName, sliceName)
	} else {
		return g.generateUnmarshalFixedValue(fieldName, sliceName)

	}
}

func (g *generateList) generateFixedMarshalValue(fieldName string) string {
	tmpl := `dst = ssz.WriteOffset(dst, offset)
offset += %s
`
	offset := g.variableSizeSSZ(fieldName)

	return fmt.Sprintf(tmpl, offset)
}

var variableSizedListTmpl = `func() int {
	s := 0
	for _, o := range {{ .FieldName }} {
		s += 4
		s += {{ .SizeComputation }}
	}
	return s
}()`

func (g *generateList) variableSizeSSZ(fieldName string) string {
	if !g.valRep.ElementValue.IsVariableSized() {
		return fmt.Sprintf("len(%s) * %d", fieldName, g.valRep.ElementValue.FixedSize())
	}

	gg := newValueGenerator(g.valRep.ElementValue, g.targetPackage)
	vslTmpl, err := template.New("variableSizedListTmpl").Parse(variableSizedListTmpl)
	if err != nil {
		panic(err)
	}
	buf := bytes.NewBuffer(nil)
	err = vslTmpl.Execute(buf, struct{
		FieldName string
		SizeComputation string
	}{
		FieldName: fieldName,
		SizeComputation: gg.variableSizeSSZ("o"),
	})
	if err != nil {
		panic(err)
	}
	return string(buf.Bytes())
}

var generateVariableMarshalValueTmpl = `if len({{ .FieldName }}) > {{ .MaxSize }} {
		return nil, ssz.ErrListTooBig
}

for _, o := range {{ .FieldName }} {
		if len(o) != {{ .ElementSize }} {
				return nil, ssz.ErrBytesLength
		}
		dst = append(dst, o) 
}`

var tmplVariableOffsetManagement = `{
	offset = 4 * len({{.FieldName}})
	for _, {{.NestedFieldName}} := range {{.FieldName}} {
		dst = ssz.WriteOffset(dst, offset)
		offset += {{.SizeComputation}}
	}
}
`

func variableOffsetManagement(vg valueGenerator, fieldName, nestedFieldName string) string {
	vomt, err := template.New("tmplVariableOffsetManagement").Parse(tmplVariableOffsetManagement)
	if err != nil {
		panic(err)
	}
	buf := bytes.NewBuffer(nil)
	err = vomt.Execute(buf, struct{
		FieldName string
		NestedFieldName string
		SizeComputation string
	}{
		FieldName: fieldName,
		NestedFieldName: nestedFieldName,
		SizeComputation: vg.variableSizeSSZ(nestedFieldName),
	})
	if err != nil {
		panic(err)
	}
	return string(buf.Bytes())
}

var tmplGenerateMarshalValueList = `if len({{.FieldName}}) > {{.MaxSize}} {
	return nil, ssz.ErrListTooBig
}
{{.OffsetManagement}}{{.MarshalValue}}`

func (g *generateList) generateVariableMarshalValue(fieldName string) string {
	mvTmpl, err := template.New("tmplGenerateMarshalValueList").Parse(tmplGenerateMarshalValueList)
	if err != nil {
		panic(err)
	}
	var marshalValue string
	var offsetMgmt string
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
		var internal string
		if g.valRep.ElementValue.IsVariableSized() {
			vm, ok := gg.(variableMarshaller)
			if !ok {
				panic(fmt.Sprintf("variable size type does not implement variableMarshaller: %v", g.valRep.ElementValue))
			}
			internal = vm.generateVariableMarshalValue(nestedFieldName)
			offsetMgmt = variableOffsetManagement(gg, fieldName, nestedFieldName)
		} else {
			internal = gg.generateFixedMarshalValue(nestedFieldName)
		}
		marshalValue = fmt.Sprintf(t, nestedFieldName, fieldName, internal)
	}
	buf := bytes.NewBuffer(nil)
	err = mvTmpl.Execute(buf, struct{
		FieldName string
		MaxSize int
		MarshalValue string
		OffsetManagement string
	}{
		FieldName: fieldName,
		MaxSize: g.valRep.MaxSize,
		MarshalValue: marshalValue,
		OffsetManagement: offsetMgmt,
	})
	if err != nil {
		panic(err)
	}
	return string(buf.Bytes())
}

var _ valueGenerator = &generateList{}
