package errcheck

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

const commonSrc = `
package p

type Inner struct {}
func (Inner) Method()

type Outer struct {Inner}
type OuterP struct {*Inner}

type InnerInterface interface {
	Method()
}

type OuterInterface interface {InnerInterface}
type MiddleInterfaceStruct struct {OuterInterface}
type OuterInterfaceStruct struct {MiddleInterfaceStruct}

var c = `

type testCase struct {
	selector   string
	expectedOk bool
	expected   []string
}

func TestWalkThroughEmbeddedInterfaces(t *testing.T) {
	cases := []testCase{
		testCase{"Inner{}.Method", false, nil},
		testCase{"(&Inner{}).Method", false, nil},
		testCase{"Outer{}.Method", false, nil},
		testCase{"InnerInterface.Method", true, []string{"test.InnerInterface"}},
		testCase{"OuterInterface.Method", true, []string{"test.OuterInterface", "test.InnerInterface"}},
		testCase{"OuterInterfaceStruct.Method", true, []string{"test.OuterInterface", "test.InnerInterface"}},
	}

	for _, c := range cases {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "test", commonSrc+c.selector, 0)
		if err != nil {
			t.Fatal(err)
		}

		conf := types.Config{}
		info := types.Info{
			Selections: make(map[*ast.SelectorExpr]*types.Selection),
		}
		_, err = conf.Check("test", fset, []*ast.File{f}, &info)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			s, ok := n.(*ast.SelectorExpr)
			if ok {
				selection, ok := info.Selections[s]
				if !ok {
					t.Fatalf("no Selection!")
				}
				ts, ok := walkThroughEmbeddedInterfaces(selection)
				if ok != c.expectedOk {
					t.Errorf("expected ok %v got %v", c.expectedOk, ok)
					return false
				}
				if !ok {
					return false
				}

				if len(ts) != len(c.expected) {
					t.Fatalf("expected %d types, got %d", len(c.expected), len(ts))
				}

				for i, e := range c.expected {
					if e != ts[i].String() {
						t.Errorf("mismatch at index %d: expected %s got %s", i, e, ts[i])
					}
				}
			}

			return true
		})

	}

}
