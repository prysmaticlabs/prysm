package builtinshadow

import (
	"errors"
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Doc explaining the tool.
const Doc = "Tool to detect declarations that shadow predeclared identifiers by having the same name."

const messageTemplate = ""

var builtins = []string{"true", "false", "iota", "nil", "append", "cap", "close", "complex", "copy", "delete", "imag",
	"len", "make", "new", "panic", "print", "println", "real", "recover", "bool", "complex128", "complex64",
	"error", "float32", "int", "int16", "int32", "int64", "int8", "rune", "string", "uint", "uint16", "uint32",
	"uint64", "uint8", "uintptr"}

// Analyzer runs static analysis.
var Analyzer = &analysis.Analyzer{
	Name:     "builtinshadow",
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
		(*ast.Ident)(nil),
	}

	inspect.Preorder(nodeFilter, func(node ast.Node) {
		identifier, ok := node.(*ast.Ident)
		if !ok {
			return
		}

		for _, builtin := range builtins {
			if identifier.Name == builtin {
				pass.Reportf(identifier.NamePos,
					"Identifier '%s' shadows a predeclared identifier with the same name. Choose another name.",
					identifier.Name)
			}
		}

		switch declaration := node.(type) {
		case *ast.FuncDecl:
			name := declaration.Name.Name
			if shadowsBuiltin(name) {

			}
		case *ast.Ident:
			name := declaration.Name
			if shadowsBuiltin(name) {
				pass.Reportf(declaration.NamePos, messageTemplate, "Identifier", name)
			}
		case *ast.TypeSpec:
			name := declaration.Name.Name
			if shadowsBuiltin(name) {
				pass.Reportf(declaration.Name.NamePos, messageTemplate, "Type", name)
			}
		}
	})

	return nil, nil
}

func shadowsBuiltin(name string) bool {

	return false
}
