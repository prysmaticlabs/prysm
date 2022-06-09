package backend

import (
	"bytes"
	"fmt"
	"github.com/prysmaticlabs/prysm/sszgen/types"
	"strings"
	"text/template"
)
var generateUnmarshalSSZTmpl = `func ({{.Receiver}} {{.Type}}) XXUnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size {{ .SizeInequality }} {{ .FixedOffset }} {
		return ssz.ErrSize
	}

	{{ .SliceDeclaration }}
{{ .ValueUnmarshaling }}
	return err
}`


func  GenerateUnmarshalSSZ(g *generateContainer) *generatedCode {
	sizeInequality := "!="
	if g.IsVariableSized() {
		sizeInequality = "<"
	}
	ums := g.unmarshalSteps()
	unmarshalBlocks := make([]string, 0)
	for i, c := range g.Contents {
		unmarshalBlocks = append(unmarshalBlocks, fmt.Sprintf("\n\t// Field %d: %s", i, c.Key))
		mg := newValueGenerator(c.Value, g.targetPackage)
		fieldName := fmt.Sprintf("%s.%s", receiverName, c.Key)

		vi, ok := mg.(valueInitializer)
		if ok {
			ini := vi.initializeValue(fieldName)
			if ini != "" {
				unmarshalBlocks = append(unmarshalBlocks, fmt.Sprintf("%s = %s", fieldName, ini))
			}
		}

		sliceName := fmt.Sprintf("s%d", i)
		mv := mg.generateUnmarshalValue(fieldName, sliceName)
		if mv != "" {
			//unmarshalBlocks = append(unmarshalBlocks, fmt.Sprintf("\t%s = %s", fieldName, mv))
			unmarshalBlocks = append(unmarshalBlocks, mv)
		}

		/*
					if !c.Value.IsVariableSized() {
						continue
					}
			_, ok := mg.(variableUnmarshaller)
			if !ok {
				continue
			}
			vm := mg.(variableUnmarshaller)
			vmc := vm.generateVariableUnmarshalValue(fieldName)
			if vmc != "" {
				unmarshalVariableBlocks = append(unmarshalVariableBlocks, fmt.Sprintf("\n\t// Field %d: %s", i, c.Key))
				unmarshalVariableBlocks = append(unmarshalVariableBlocks, "\t" + vmc)
			}
		*/
	}

	sliceDeclarations := strings.Join([]string{ums.fixedSlices(), "", ums.variableSlices(g.fixedOffset())}, "\n")
	unmTmpl, err := template.New("GenerateUnmarshalSSZTmpl").Parse(generateUnmarshalSSZTmpl)
	if err != nil {
		panic(err)
	}
	buf := bytes.NewBuffer(nil)
	err = unmTmpl.Execute(buf, struct{
		Receiver string
		Type string
		SizeInequality string
		FixedOffset int
		SliceDeclaration string
		ValueUnmarshaling string
	}{
		Receiver: receiverName,
		Type: fmt.Sprintf("*%s", g.TypeName()),
		SizeInequality: sizeInequality,
		FixedOffset: g.fixedOffset(),
		SliceDeclaration: sliceDeclarations,
		ValueUnmarshaling: strings.Join(unmarshalBlocks, "\n"),
	})
	// TODO: allow GenerateUnmarshalSSZ to return an error since template.Execute
	// can technically return an error
	if err != nil {
		panic(err)
	}
	return &generatedCode{
		blocks:  []string{string(buf.Bytes())},
		imports: extractImportsFromContainerFields(g.Contents, g.targetPackage),
	}
}

type unmarshalStep struct {
	valRep types.ValRep
	fieldNumber int
	fieldName string
	beginByte int
	endByte int
	previousVariable *unmarshalStep
	nextVariable *unmarshalStep
}

type unmarshalStepSlice []*unmarshalStep

func (us *unmarshalStep) fixedSize() int {
	return us.valRep.FixedSize()
}

func (us *unmarshalStep) variableOffset(outerFixedSize int) string {
	o := fmt.Sprintf("v%d := ssz.ReadOffset(buf[%d:%d]) // %s", us.fieldNumber, us.beginByte, us.endByte, us.fieldName)
	if us.previousVariable == nil {
		o += fmt.Sprintf("\nif v%d < %d {\n\treturn ssz.ErrInvalidVariableOffset\n}", us.fieldNumber, outerFixedSize)
		o += fmt.Sprintf("\nif v%d > size {\n\treturn ssz.ErrOffset\n}", us.fieldNumber)
	} else {
		o += fmt.Sprintf("\nif v%d > size || v%d < v%d {\n\treturn ssz.ErrOffset\n}", us.fieldNumber, us.fieldNumber, us.previousVariable.fieldNumber)
	}
	return o
}

func (us *unmarshalStep) slice() string {
	if us.valRep.IsVariableSized() {
		if us.nextVariable == nil {
			return fmt.Sprintf("s%d := buf[v%d:]\t\t// %s", us.fieldNumber, us.fieldNumber, us.fieldName)
		}
		return fmt.Sprintf("s%d := buf[v%d:v%d]\t\t// %s", us.fieldNumber, us.fieldNumber, us.nextVariable.fieldNumber, us.fieldName)
	}
	return fmt.Sprintf("s%d := buf[%d:%d]\t\t// %s", us.fieldNumber, us.beginByte, us.endByte, us.fieldName)
}

func (steps unmarshalStepSlice) fixedSlices() string {
	slices := make([]string, 0)
	for _, s := range steps {
		if s.valRep.IsVariableSized() {
			continue
		}
		slices = append(slices, s.slice())
	}
	return strings.Join(slices, "\n")
}

func (steps unmarshalStepSlice)  variableSlices(outerSize int) string {
	validate := make([]string, 0)
	assign := make([]string, 0)
	for _, s := range steps {
		if !s.valRep.IsVariableSized() {
			continue
		}
		validate = append(validate, s.variableOffset(outerSize))
		assign = append(assign, s.slice())
	}
	return strings.Join(append(validate, assign...), "\n")
}

func (g *generateContainer) unmarshalSteps() unmarshalStepSlice{
	ums := make([]*unmarshalStep, 0)
	var begin, end int
	var prevVariable *unmarshalStep
	for i, c := range g.Contents {
		begin = end
		end += c.Value.FixedSize()
		um := &unmarshalStep{
			valRep: c.Value,
			fieldNumber: i,
			fieldName: fmt.Sprintf("%s.%s", receiverName, c.Key),
			beginByte: begin,
			endByte: end,
		}
		if c.Value.IsVariableSized() {
			if prevVariable != nil {
				um.previousVariable = prevVariable
				prevVariable.nextVariable = um
			}
			prevVariable = um
		}

		ums = append(ums, um)
	}
	return ums
}