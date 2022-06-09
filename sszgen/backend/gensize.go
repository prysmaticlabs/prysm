package backend

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

var sizeBodyTmpl = `func ({{.Receiver}} {{.Type}}) XXSizeSSZ() (int) {
	size := {{.FixedSize}}
	{{- .VariableSize }}
	return size
}`

func GenerateSizeSSZ(g *generateContainer) *generatedCode {
	sizeTmpl, err := template.New("GenerateSizeSSZ").Parse(sizeBodyTmpl)
	if err != nil {
		panic(err)
	}
	buf := bytes.NewBuffer(nil)

	fixedSize := 0
	variableComputations := make([]string, 0)
	for _, c := range g.Contents {
		vg := newValueGenerator(c.Value, g.targetPackage)
		fixedSize += c.Value.FixedSize()
		if !c.Value.IsVariableSized() {
			continue
		}
		fieldName := fmt.Sprintf("%s.%s", receiverName, c.Key)
		vi, ok := vg.(valueInitializer)
		if ok {
			ini := vi.initializeValue(fieldName)
			if ini != "" {
				variableComputations = append(variableComputations, fmt.Sprintf("if %s == nil {\n\t%s = %s\n}", fieldName, fieldName, ini))
			}
		}
		cv := vg.variableSizeSSZ(fieldName)
		if cv != "" {
			variableComputations = append(variableComputations, fmt.Sprintf("\tsize += %s", cv))
		}
	}

	err = sizeTmpl.Execute(buf, struct{
		Receiver string
		Type string
		FixedSize int
		VariableSize string
	}{
		Receiver: receiverName,
		Type: fmt.Sprintf("*%s", g.TypeName()),
		FixedSize: fixedSize,
		VariableSize: "\n" + strings.Join(variableComputations, "\n"),
	})
	// TODO: allow GenerateSizeSSZ to return an error since template.Execute
	// can technically return an error
	if err != nil {
		panic(err)
	}
	return &generatedCode{
		blocks:  []string{string(buf.Bytes())},
		imports: extractImportsFromContainerFields(g.Contents, g.targetPackage),
	}
}