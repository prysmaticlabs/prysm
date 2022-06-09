package sszgen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"

	"github.com/prysmaticlabs/prysm/sszgen/types"
	"golang.org/x/tools/go/packages"
)

type ParseNode struct {
	PackagePath    string
	Name           string
	typeSpec *ast.TypeSpec
	typeExpression ast.Expr
	FileParser FileParser
	PackageParser  PackageParser
	ValRep         types.ValRep
	Tag string
}

func (pn *ParseNode) DeclarationRef() *DeclarationRef {
	return &DeclarationRef{Name: pn.Name, Package: pn.PackagePath}
}

type DeclarationRef struct {
	Name string
	Package string
}

func (ts *ParseNode) TypeExpression() ast.Expr {
	if ts.typeSpec != nil {
		return ts.typeSpec.Type
	}
	if ts.typeExpression != nil {
		return ts.typeExpression
	}
	return nil
}

type FileParser interface {
	ResolveAlias(string) (string, error)
}

type astFileParser struct {
	file *ast.File
	filename string
}

var _ FileParser = &astFileParser{}

func (afp *astFileParser) ResolveAlias(alias string) (string, error) {
	for _, imp := range afp.file.Imports {
		if imp.Name.Name == alias {
			resolved, err := strconv.Unquote(imp.Path.Value)
			return resolved, err
		}
	}
	return "", fmt.Errorf("Could not resolve alias %s from filename '%s'", alias, afp.filename)
}

type PackageParser interface {
	Imports() ([]*ast.ImportSpec, error)
	AllParseNodes() []*ParseNode
	GetType(name string) (*ParseNode, error)
	Path() string // parser's package path
	PackageName() (string, error) // "real" name ie `package $NAME` declaration in source files in package
}

type packageParser struct {
	packagePath string
	files map[string]*ast.File
}

func (pp *packageParser) Imports() ([]*ast.ImportSpec, error) {
	imports := make([]*ast.ImportSpec, 0)
	for _, f := range pp.files {
		for _, imp := range f.Imports {
			imports = append(imports, imp)
		}
	}
	return imports, nil
}

func (pp *packageParser) AllParseNodes() []*ParseNode {
	structs := make([]*ParseNode, 0)
	for fname, f := range pp.files {
		for name, obj := range f.Scope.Objects {
			if obj.Kind != ast.Typ {
				continue
			}
			typeSpec, ok := obj.Decl.(*ast.TypeSpec)
			if !ok {
				continue
			}
			ts := &ParseNode{
				Name:           name,
				//TypeExpression: typeSpec.Type,
				typeSpec: typeSpec,
				FileParser:           &astFileParser{filename: fname, file:f},
				PackagePath:    pp.packagePath,
			}
			structs = append(structs, ts)
		}
	}
	return structs
}

func (pp *packageParser) PackageName() (string, error) {
	for _, f := range pp.files {
		return f.Name.Name, nil
	}
	return "", fmt.Errorf("Could not determine package name for package path %s", pp.packagePath)
}

func (pp *packageParser) GetType(name string) (*ParseNode, error) {
	for fname, f := range pp.files {
		for objName, obj := range f.Scope.Objects {
			if obj.Kind != ast.Typ {
				continue
			}
			typeSpec, ok := obj.Decl.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if name == objName {
				return &ParseNode{
					Name:           objName,
					//TypeExpression: typeSpec.Type,
					typeSpec: typeSpec,
					FileParser:           &astFileParser{file: f, filename: fname},
					PackageParser:  pp,
				}, nil
			}
		}
	}
	return nil, fmt.Errorf("Could not find struct named '%s' in package %s", name, pp.packagePath)
}

func (pp *packageParser) Path() string {
	return pp.packagePath
}

func NewPackageParser(packageName string) (*packageParser, error) {
	cfg := &packages.Config{Mode: packages.NeedFiles | packages.NeedSyntax}
	pkgs, err := packages.Load(cfg, []string{packageName}...)
	if err != nil {
		return nil, err
	}
	for _, pkg := range pkgs {
		if pkg.ID != packageName {
			continue
		}
		pp := &packageParser{packagePath: pkg.ID, files: make(map[string]*ast.File)}
		for _, f := range pkg.GoFiles {
			syn, err := parser.ParseFile(token.NewFileSet(), f, nil, parser.AllErrors)
			if err != nil {
				return nil, err
			}
			pp.files[f] = syn
		}
		return pp, nil
	}
	return nil, fmt.Errorf("Package named '%s' could not be loaded from the go build system. Please make sure the current folder contains the go.mod for the target package, or that its go.mod is in a parent directory", packageName)
}

