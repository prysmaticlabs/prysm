package main

import (
	_ "embed"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/logrusorgru/aurora"
	"github.com/urfave/cli/v2"
)

//go:embed data
var specText string

// Regex to find Python's "def".
var m = regexp.MustCompile("def\\s(.*)\\(.*")

var (
	dirFlag = &cli.StringFlag{
		Name:     "dir",
		Value:    "",
		Usage:    "Path to a directory containing Golang files to check",
		Required: true,
	}
	au = aurora.NewAurora(true /* enable colors */)
)

func main() {
	app := &cli.App{
		Name:        "Specs checker utility",
		Description: "Checks that specs pseudo code used in comments is up to date",
		Usage:       "helps keeping specs pseudo code up to date!",
		Commands: []*cli.Command{
			{
				Name:  "check",
				Usage: "Checks that all doc strings",
				Flags: []cli.Flag{
					dirFlag,
				},
				Action: check,
			},
			{
				Name:   "download",
				Usage:  "Downloads the latest specs docs",
				Action: download,
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func check(cliCtx *cli.Context) error {
	// Obtain reference snippets.
	defs, err := parseSpecs(specText)
	if err != nil {
		return err
	}

	// Walk the path, and process all contained Golang files.
	fileWalker := func(path string, info os.FileInfo, err error) error {
		if info == nil {
			return fmt.Errorf("invalid input dir %q", path)
		}
		if !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}
		if err := inspectFile(path, defs); err != nil {
			return err
		}
		return nil
	}
	if err := filepath.Walk(cliCtx.String(dirFlag.Name), fileWalker); err != nil {
		return err
	}

	return nil
}

func inspectFile(path string, defs map[string][]string) error {
	// Parse source files, and check the pseudo code.
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	ast.Inspect(file, func(node ast.Node) bool {
		switch stmt := node.(type) {
		case *ast.CommentGroup:
			// Ignore comment groups that do not have python pseudo-code.
			chunk := stmt.Text()
			if !m.MatchString(chunk) {
				return true
			}

			pos := fset.Position(node.Pos())

			// Trim the chunk, so that it starts from Python's "def".
			loc := m.FindStringIndex(chunk)
			chunk = chunk[loc[0]:]

			// Find out Python function name.
			defName, defBody := parseDefChunk(chunk)
			if defName == "" {
				fmt.Printf("%s: cannot parse comment pseudo code\n", pos)
				return false
			}

			// Calculate differences with reference implementation.
			refDefs, ok := defs[defName]
			if !ok {
				fmt.Printf("%s: %q is not found in spec docs\n", pos, defName)
				return false
			}
			if !matchesRefImplementation(defName, refDefs, defBody) {
				fmt.Printf("%s: %q code does not match reference implementation in specs\n", pos, defName)
				return false
			}
		}
		return true
	})

	return nil
}

func download(cliCtx *cli.Context) error {
	return nil
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
