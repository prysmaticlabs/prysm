package analyzers

import (
	"errors"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// GetInspector returns the inspector obtained from passing through inspect.Analyzer
func GetInspector(pass *analysis.Pass) (*inspector.Inspector, error) {
	inspector, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, errors.New("analyzer is not type *inspector.Inspector")
	}
	return inspector, nil
}
