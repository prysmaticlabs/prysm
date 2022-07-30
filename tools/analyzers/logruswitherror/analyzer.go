// Package logruswitherror implements a static analyzer to ensure that log statements do not use
// errors in templated log statements. Authors should use logrus.WithError().
package logruswitherror

import (
	"errors"
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Doc explaining the tool.
const Doc = "TODO"

const errImproperUsage = "use log.WithError rather than templated log statements with errors"

// List of logrus templated log functions.
var logFns = map[string]interface{}{
	"Debugf":   nil,
	"Infof":    nil,
	"Printf":   nil,
	"Warnf":    nil,
	"Warningf": nil,
	"Errorf":   nil,
	"Fatalf":   nil,
	"Panicf":   nil,
}

// Analyzer runs static analysis.
var Analyzer = &analysis.Analyzer{
	Name:     "logruswitherror",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, errors.New("analyzer is not type *inspector.Inspector")
	}

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	_, _ = inspect, nodeFilter

	inspect.Preorder(nodeFilter, func(node ast.Node) {
		switch stmt := node.(type) {
		case *ast.CallExpr:
			// Lookup function name
			fnName := stmt.Fun.(*ast.SelectorExpr).Sel.Name

			// If function matches any of the logrus functions, check if it uses errors.
			if _, ok := logFns[fnName]; !ok {
				return
			}

			for i, arg := range stmt.Args {
				if i < 1 {
					continue
				}

				// Check CallExpr and Ident for error type.
				switch a := arg.(type) {
				case *ast.CallExpr:
					// _ = a.Fun.(*ast.SelectorExpr).Sel

					// if err := ast.Print(pass.Fset, a); err != nil {
					// 	panic(err)
					// }
					// return
				case *ast.Ident:
					// Check if the error is a variable.

					f, ok := a.Obj.Decl.(*ast.Field)
					if !ok {
						panic("its not ok 0")
					}
					typ, ok := f.Type.(*ast.Ident)
					if !ok {
						panic("its not ok 1")
					}

					if typ.Name == "error" {
						pass.Reportf(a.Pos(), errImproperUsage)
					}
				}
			}
		}
	})

	return nil, nil // TODO
}
