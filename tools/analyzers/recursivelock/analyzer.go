package recursivelock

import (
	"errors"
	"fmt"

	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Doc explaining the tool
const Doc = "Tool to check for recursive or nested mutex read lock calls"

// Analyzer runs static analysis.
var Analyzer = &analysis.Analyzer{
	Name:     "recursivelock",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

var errNestedRLock = errors.New("found recursive read lock call")

func run(pass *analysis.Pass) (interface{}, error) {
	inspect, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, errors.New("analyzer is not type *inspector.Inspector")
	}

	// filters out other pieces of source code except for function/method calls
	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
		(*ast.DeferStmt)(nil),
		(*ast.FuncDecl)(nil),
	}

	foundRLock := 0
	deferredRLock := false
	endPos := token.NoPos
	inspect.Preorder(nodeFilter, func(node ast.Node) {
		if node.Pos() > endPos && deferredRLock { // deferred RUnlocks are counted at the end
			deferredRLock = false
			foundRLock--
		}
		switch stmt := node.(type) {
		case *ast.CallExpr:
			name := getName(stmt.Fun)
			if name == "RLock" { // if the method found is an RLock method
				if foundRLock > 0 { // if we have already seen an RLock method without seeing a corresponding RUnlock
					pass.Reportf(
						node.Pos(),
						fmt.Sprintf(
							"%v",
							errNestedRLock,
						),
					)
				}
				foundRLock++
			} else if name == "RUnlock" && !deferredRLock {
				foundRLock--
			} else if name != "RUnlock" && foundRLock > 0 {
				if stack := hasNestedRLock(name, inspect, pass.Fset); stack != "" { // hasNestedRLock returns a stack showing where the nested RLock exactly is; however, I did not know how to include this in the test diagnostic
					pass.Reportf(
						node.Pos(),
						fmt.Sprintf(
							"%v",
							errNestedRLock,
							// stack,
						),
					)
				}
			}
		case *ast.DeferStmt:
			name := getName(stmt.Call.Fun)
			if name == "RUnlock" {
				deferredRLock = true
			}
		case *ast.FuncDecl:
			endPos = stmt.Body.Rbrace
		}
	})

	return nil, nil
}

// gets the name of a call expression whether it is a dot statement or not
func getName(fun ast.Expr) string {
	switch expr := fun.(type) {
	case *ast.SelectorExpr:
		return expr.Sel.Name
	case *ast.Ident:
		return expr.Name
	}
	return ""
}

// recursively checks called functions to determine whether they call RLock()
func hasNestedRLock(funcName string, inspect *inspector.Inspector, f *token.FileSet) (retStack string) {
	node := findCallDeclarationNode(funcName, inspect)
	if node == nil {
		return ""
	}
	ast.Inspect(node, func(iNode ast.Node) bool {
		switch stmt := iNode.(type) {
		case *ast.CallExpr:
			name := getName(stmt.Fun)
			addition := fmt.Sprintf("\tat %v\n", f.Position(iNode.Pos()))
			if name == "RLock" { // if the method found is an RLock method
				retStack += addition
			} else if name != "RUnlock" && name != funcName { // name should not equal the previousName to prevent infinite recursive loop
				stack := hasNestedRLock(name, inspect, f)
				if stack != "" {
					retStack += addition + stack
				}
			}
		}
		return true
	})
	return retStack
}

// finds the call declaration node of name targetName
func findCallDeclarationNode(targetName string, inspect *inspector.Inspector) ast.Node {
	var retNode ast.Node = nil
	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}
	inspect.Preorder(nodeFilter, func(node ast.Node) {
		funcDec, _ := node.(*ast.FuncDecl)
		name := funcDec.Name.Name
		if targetName == name {
			retNode = node
		}
	})
	return retNode
}
