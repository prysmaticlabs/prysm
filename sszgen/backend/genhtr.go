package backend

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// ChunkSize is used to check if packed bytes align to the chunk sized used by the
// merkleization algorithm. If not, the bytes should be zero-padded to the
// nearest multiple of ChunkSize.
const ChunkSize = 32

var htrTmpl = `func ({{.Receiver}} {{.Type}}) XXHashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := {{.Receiver}}.XXHashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func ({{.Receiver}} {{.Type}}) XXHashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	{{.HTRSteps}}
	hh.Merkleize(indx)
	return nil
}`

func GenerateHashTreeRoot(g *generateContainer) *generatedCode {
	htrTmpl, err := template.New("GenerateHashTreeRoot").Parse(htrTmpl)
	if err != nil {
		panic(err)
	}
	buf := bytes.NewBuffer(nil)
	htrSteps := make([]string, 0)
	for i, c := range g.Contents {
		fieldName := fmt.Sprintf("%s.%s", receiverName, c.Key)
		htrSteps = append(htrSteps, fmt.Sprintf("\t// Field %d: %s", i, c.Key))
		vg := newValueGenerator(c.Value, g.targetPackage)
		htrp, ok := vg.(htrPutter)
		if !ok {
			continue
		}
		htrSteps = append(htrSteps, htrp.generateHTRPutter(fieldName))
	}
	err  = htrTmpl.Execute(buf, struct{
		Receiver string
		Type string
		HTRSteps string
	}{
		Receiver: receiverName,
		Type: fmt.Sprintf("*%s", g.TypeName()),
		HTRSteps: strings.Join(htrSteps, "\n"),
	})
	if err != nil {
		panic(err)
	}
	// TODO: allow GenerateHashTreeRoot to return an error since template.Execute
	// can technically return an error (get rid of the panics)
	return &generatedCode{
		blocks:  []string{string(buf.Bytes())},
		imports: extractImportsFromContainerFields(g.Contents, g.targetPackage),
	}
}