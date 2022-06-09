package backend

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

var marshalBodyTmpl = `func ({{.Receiver}} {{.Type}}) XXMarshalSSZ() ([]byte, error) {
	buf := make([]byte, {{.Receiver}}.XXSizeSSZ())
	return {{.Receiver}}.XXMarshalSSZTo(buf[:0])
}

func ({{.Receiver}} {{.Type}}) XXMarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
{{- .OffsetDeclaration -}}
{{- .ValueMarshaling }}
{{- .VariableValueMarshaling }}
	return dst, err
}`

func GenerateMarshalSSZ(g *generateContainer) *generatedCode {
	sizeTmpl, err := template.New("GenerateMarshalSSZ").Parse(marshalBodyTmpl)
	if err != nil {
		panic(err)
	}
	buf := bytes.NewBuffer(nil)

	marshalValueBlocks := make([]string, 0)
	marshalVariableValueBlocks := make([]string, 0)
	offset := 0
	for i, c := range g.Contents {
		// only lists need the offset variable
		mg := newValueGenerator(c.Value, g.targetPackage)
		fieldName := fmt.Sprintf("%s.%s", receiverName, c.Key)
		marshalValueBlocks = append(marshalValueBlocks, fmt.Sprintf("\n\t// Field %d: %s", i, c.Key))
		vi, ok := mg.(valueInitializer)
		if ok {
			ini := vi.initializeValue(fieldName)
			if ini != "" {
				marshalValueBlocks = append(marshalValueBlocks , fmt.Sprintf("if %s == nil {\n\t%s = %s\n}", fieldName, fieldName, ini))
			}
		}
		mv := mg.generateFixedMarshalValue(fieldName)
		marshalValueBlocks = append(marshalValueBlocks, "\t" + mv)
		offset += c.Value.FixedSize()
		if !c.Value.IsVariableSized() {
			continue
		}
		vm, ok := mg.(variableMarshaller)
		if !ok {
			continue
		}
		vmc := vm.generateVariableMarshalValue(fieldName)
		if vmc != "" {
			marshalVariableValueBlocks = append(marshalVariableValueBlocks, fmt.Sprintf("\n\t// Field %d: %s", i, c.Key))
			marshalVariableValueBlocks = append(marshalVariableValueBlocks, "\t" + vmc)
		}
	}
	// only set the offset declaration if we need it
	// otherwise we'll have an unused variable (syntax error)
	offsetDeclaration := ""
	if g.IsVariableSized() {
		// if there are any variable sized values in the container, we'll need to set this offset declaration
		// so it gets rendered to the top of the marshal method
		offsetDeclaration = fmt.Sprintf("\noffset := %d\n", offset)
	}

	err = sizeTmpl.Execute(buf, struct{
		Receiver string
		Type string
		OffsetDeclaration string
		ValueMarshaling string
		VariableValueMarshaling string
	}{
		Receiver: receiverName,
		Type: fmt.Sprintf("*%s", g.TypeName()),
		OffsetDeclaration: offsetDeclaration,
		ValueMarshaling: "\n" + strings.Join(marshalValueBlocks, "\n"),
		VariableValueMarshaling: "\n" + strings.Join(marshalVariableValueBlocks, "\n"),
	})
	// TODO: allow GenerateMarshalSSZ to return an error since template.Execute
	// can technically return an error
	if err != nil {
		panic(err)
	}
	return &generatedCode{
		blocks:  []string{string(buf.Bytes())},
		imports: extractImportsFromContainerFields(g.Contents, g.targetPackage),
	}
}
