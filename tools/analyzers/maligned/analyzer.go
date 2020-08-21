package maligned

import (
	"go/ast"
	"go/types"

	"github.com/prysmaticlabs/prysm/tools/analyzers"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

// Doc explaining the tool.
const Doc = "Tool to detect Go structs that would take less memory if their fields were sorted."

// Analyzer runs static analysis.
var Analyzer = &analysis.Analyzer{
	Name:     "maligned",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspector, err := analyzers.GetInspector(pass)
	if err != nil {
		return nil, err
	}

	nodeFilter := []ast.Node{
		(*ast.StructType)(nil),
	}

	inspector.Preorder(nodeFilter, func(node ast.Node) {
		if s, ok := node.(*ast.StructType); ok {
			if err := malign(node.Pos(), pass.TypesInfo.Types[s].Type.(*types.Struct)); err != nil {
				pass.Reportf(node.Pos(), err.Error())
			}
		}
	})

	return nil, nil
}
