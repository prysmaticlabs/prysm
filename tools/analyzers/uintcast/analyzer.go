package uintcast

import (
	"errors"
	"go/ast"
	"go/types"
	"strings"

	"github.com/gostaticanalysis/comment"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Doc explaining the tool.
const Doc = "Ensure that uint variables are not cast improperly where the value could overflow. " +
	"This check can be suppressed with the `lint:ignore uintcast` comment with proper justification."

// Analyzer runs static analysis.
var Analyzer = &analysis.Analyzer{
	Name:     "uintcast",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspection, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, errors.New("analyzer is not type *inspector.Inspector")
	}

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	commentMap := comment.New(pass.Fset, pass.Files)

	inspection.Preorder(nodeFilter, func(node ast.Node) {
		cg := commentMap.CommentsByPosLine(pass.Fset, node.Pos())
		for _, c := range cg {
			if strings.Contains(c.Text(), "lint:ignore uintcast") {
				return
			}
		}

		switch node := node.(type) {
		case *ast.CallExpr:
			// Cast/conversion calls have one argument and no ellipsis.
			if len(node.Args) != 1 || node.Ellipsis.IsValid() {
				return
			}

			if arg, ok := node.Args[0].(*ast.Ident); ok {
				if typ, ok := pass.TypesInfo.Types[arg].Type.(*types.Basic); ok {
					// Ignore types that are not uint variants.
					if typ.Kind() < types.Uint || typ.Kind() > types.Uint64 {
						return
					}
					if fnTyp, ok := pass.TypesInfo.Types[node.Fun].Type.(*types.Basic); ok {
						if fnTyp.Kind() >= types.Int && fnTyp.Kind() <= types.Int64 {
							pass.Reportf(arg.Pos(), "Unsafe cast from %s to %s.", typ, fnTyp)
						}
					}
				}

			}
		}
	})

	return nil, nil
}
