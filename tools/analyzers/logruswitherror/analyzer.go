// Package logruswitherror implements a static analyzer to ensure that log statements do not use
// errors in templated log statements. Authors should use logrus.WithError().
package logruswitherror

import (
	"errors"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

// Doc explaining the tool.
const Doc = "TODO"

var errWeakCrypto = errors.New("use logrus.WithError(err) rather than templated log statements")

// Analyzer runs static analysis.
var Analyzer = &analysis.Analyzer{
	Name:     "logruswitherror",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	return nil, nil // TODO
}
