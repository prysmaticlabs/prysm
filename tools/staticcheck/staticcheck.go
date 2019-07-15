package staticcheck

import (
	"golang.org/x/tools/go/analysis"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
)

var analyzers []*analysis.Analyzer

func init() {
	for _, v := range simple.Analyzers {
		analyzers = append(analyzers, v)
	}
	for _, v := range staticcheck.Analyzers {
		analyzers = append(analyzers, v)
	}
	for _, v := range stylecheck.Analyzers {
		analyzers = append(analyzers, v)
	}
}

var Analyzer = &analysis.Analyzer{
	Name: "staticcheck",
	Doc:  "reports result of github.com/dominikh/go-tools/cmd/staticcheck",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, a := range analyzers {
		if r, err := a.Run(pass); err != nil {
			return r, err
		}
	}
	return nil, nil
}
