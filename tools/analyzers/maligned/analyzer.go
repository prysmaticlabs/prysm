package maligned

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const Doc = "Tool to detect Go structs that would take less memory if their fields were sorted."

var Analyzer = &analysis.Analyzer{
	Name:     "maligned",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.StructType)(nil),
	}

	inspect.Preorder(nodeFilter, func(node ast.Node) {
		if s, ok := node.(*ast.StructType); ok {
			if err := malign(node.Pos(), pass.TypesInfo.Types[s].Type.(*types.Struct)); err != nil {
				pass.Reportf(node.Pos(), err.Error())
			}
		}
	})

	return nil, nil
}
