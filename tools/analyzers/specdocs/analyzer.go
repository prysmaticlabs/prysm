// Package specdocs implements a static analyzer to ensure that pseudo code we use in our comments, when implementing
// functions defined in specs, is up to date. Reference specs documentation is cached (so that we do not need to
// download it every time build is run).
package specdocs

import (
	_ "embed"
	"errors"
	"go/ast"
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

//go:embed data
var specText string

// Regex to find Python's "def".
var m = regexp.MustCompile("def\\s(.*)\\(.*")

// Doc explaining the tool.
const Doc = "Tool to enforce that specs pseudo code is up to date"

// Analyzer runs static analysis.
var Analyzer = &analysis.Analyzer{
	Name:     "specdocs",
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
		(*ast.CommentGroup)(nil),
	}

	// Obtain reference snippets.
	defs, err := parseSpecs(specText)
	if err != nil {
		return nil, err
	}

	inspection.Preorder(nodeFilter, func(node ast.Node) {
		switch stmt := node.(type) {
		case *ast.CommentGroup:
			// Ignore comment groups that do not have python pseudo-code.
			chunk := stmt.Text()
			if !m.MatchString(chunk) {
				return
			}

			// Trim the chunk, so that it starts from Python's "def".
			loc := m.FindStringIndex(chunk)
			chunk = chunk[loc[0]:]

			// Find out Python function name.
			defName, defBody := parseDefChunk(chunk)
			if defName == "" {
				pass.Reportf(node.Pos(), "cannot parse comment pseudo code")
				return
			}

			// Calculate differences with reference implementation.
			refDefs, ok := defs[defName]
			if !ok {
				pass.Reportf(node.Pos(), "%q is not found in spec docs", defName)
				return
			}
			if !matchesRefImplementation(defName, refDefs, defBody) {
				pass.Reportf(node.Pos(), "%q code doesn not match reference implementation in specs", defName)
				return
			}
		}
	})

	return nil, nil
}

// parseSpecs parses input spec docs into map of function name -> array of function bodies
// (single entity may have several definitions).
func parseSpecs(input string) (map[string][]string, error) {
	chunks := strings.Split(strings.ReplaceAll(input, "```python", ""), "```")
	defs := make(map[string][]string, len(chunks))
	for _, chunk := range chunks {
		defName, defBody := parseDefChunk(chunk)
		if defName == "" {
			continue
		}
		defs[defName] = append(defs[defName], defBody)
	}
	return defs, nil
}

func parseDefChunk(chunk string) (string, string) {
	chunk = strings.TrimLeft(chunk, "\n")
	if chunk == "" {
		return "", ""
	}
	chunkLines := strings.Split(chunk, "\n")
	// Ignore all snippets, that do not define functions.
	if chunkLines[0][:4] != "def " {
		return "", ""
	}
	defMatches := m.FindStringSubmatch(chunkLines[0])
	if len(defMatches) < 2 {
		return "", ""
	}
	return strings.Trim(defMatches[1], " "), chunk
}

// matchesRefImplementation compares input string to reference code snippets (there might be multiple implementations).
func matchesRefImplementation(defName string, refDefs []string, input string) bool {
	for _, refDef := range refDefs {
		refDefLines := strings.Split(refDef, "\n")
		inputLines := strings.Split(input, "\n")

		matchesPerfectly := true
		for i := 0; i < len(refDefs); i++ {
			a, b := strings.Trim(refDefLines[i], " "), strings.Trim(inputLines[i], " ")
			if a != b {
				matchesPerfectly = false
				break
			}
		}
		if matchesPerfectly {
			return true
		}
	}
	return false
}
