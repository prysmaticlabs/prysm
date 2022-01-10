// Analyzer tool for detecting nested or recursive mutex read lock statements

package recursivelock

import (
	"errors"
	"fmt"

	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

// Analyzer runs static analysis.
var Analyzer = &analysis.Analyzer{
	Name:     "recursivelock",
	Doc:      "Checks for recursive or nested RLock calls",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

var errNestedRLock = errors.New("found recursive read lock call")

func run(pass *analysis.Pass) (interface{}, error) {
	inspect, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, errors.New("analyzer is not type *inspector.Inspector")
	}

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
		(*ast.DeferStmt)(nil),
		(*ast.FuncDecl)(nil),
		(*ast.FuncLit)(nil),
		(*ast.File)(nil),
		(*ast.ReturnStmt)(nil),
	}

	var keepTrackOf tracker
	inspect.Preorder(nodeFilter, func(node ast.Node) {
		if keepTrackOf.funcLitEnd.IsValid() && node.Pos() <= keepTrackOf.funcLitEnd {
			return
		} else {
			keepTrackOf.funcLitEnd = token.NoPos
		}
		if keepTrackOf.deferEnd.IsValid() && node.Pos() > keepTrackOf.deferEnd {
			keepTrackOf.deferEnd = token.NoPos
		} else if keepTrackOf.deferEnd.IsValid() {
			return
		}
		if keepTrackOf.retEnd.IsValid() && node.Pos() > keepTrackOf.retEnd {
			keepTrackOf.retEnd = token.NoPos
			keepTrackOf.incFRU()
		}

		switch stmt := node.(type) {
		case *ast.CallExpr:
			call := getCallInfo(pass.TypesInfo, stmt)
			if call == nil {
				break
			}
			name := call.name
			selMap := mapSelTypes(stmt, pass)
			if selMap == nil {
				break
			}
			if keepTrackOf.rLockSelector != nil {
				if keepTrackOf.foundRLock > 0 {
					if keepTrackOf.foundRLock > 0 && keepTrackOf.rLockSelector.isEqual(selMap, 0) {
						pass.Reportf(
							node.Pos(),
							fmt.Sprintf(
								"%v",
								errNestedRLock,
							),
						)
					} else if keepTrackOf.foundRLock > 0 {
						if stack := hasNestedRLock(keepTrackOf.rLockSelector, selMap, call, inspect, pass, make(map[string]bool)); stack != "" {
							pass.Reportf(
								node.Pos(),
								fmt.Sprintf(
									"%v\n%v",
									errNestedRLock,
									stack,
								),
							)
						}
					}
				}
				if name == "RUnlock" && keepTrackOf.rLockSelector.isEqual(selMap, 1) {
					keepTrackOf.deincFRU()
				}
			} else if name == "RLock" && keepTrackOf.foundRLock == 0 {
				keepTrackOf.rLockSelector = selMap
				keepTrackOf.incFRU()
			}

		case *ast.File:
			keepTrackOf = tracker{}

		case *ast.FuncDecl:
			keepTrackOf = tracker{}
			keepTrackOf.funcEnd = stmt.End()

		case *ast.FuncLit:
			if keepTrackOf.funcLitEnd == token.NoPos {
				keepTrackOf.funcLitEnd = stmt.End()
			}

		case *ast.DeferStmt:
			call := getCallInfo(pass.TypesInfo, stmt.Call)
			if keepTrackOf.deferEnd == token.NoPos {
				keepTrackOf.deferEnd = stmt.End()
			}
			if call != nil && call.name == "RUnlock" {
				keepTrackOf.deferredRUnlock = true
			}

		case *ast.ReturnStmt:
			if keepTrackOf.deferredRUnlock && keepTrackOf.retEnd == token.NoPos {
				keepTrackOf.deincFRU()
				keepTrackOf.retEnd = stmt.End()
			}
		}
	})
	return nil, nil
}

type tracker struct {
	funcEnd         token.Pos
	retEnd          token.Pos
	deferEnd        token.Pos
	funcLitEnd      token.Pos
	deferredRUnlock bool
	foundRLock      int
	rLockSelector   *selIdentList
}

func (t tracker) String() string {
	return fmt.Sprintf("funcEnd:%v\nretEnd:%v\ndeferEnd:%v\ndeferredRU:%v\nfoundRLock:%v\n", t.funcEnd, t.retEnd, t.deferEnd, t.deferredRUnlock, t.foundRLock)
}

func (t *tracker) deincFRU() {
	if t.foundRLock > 0 {
		t.foundRLock -= 1
	}
}
func (t *tracker) incFRU() {
	t.foundRLock += 1
}

type selIdentNode struct {
	next   *selIdentNode
	this   *ast.Ident
	typObj types.Object
}

type selIdentList struct {
	start        *selIdentNode
	length       int
	current      *selIdentNode
	currentIndex int
}

func (s *selIdentList) next() (n *selIdentNode) {
	n = s.current.next
	if n != nil {
		s.current = n
		s.currentIndex++
	}
	return n
}

func (s *selIdentList) reset() {
	s.current = s.start
	s.currentIndex = 0
}

func (s *selIdentList) isEqual(s2 *selIdentList, offset int) bool {
	if s2 == nil || (s.length != s2.length) {
		return false
	}
	s.reset()
	s2.reset()
	for i := true; i; {
		if !s.current.isEqual(s2.current) {
			return false
		}
		if s.currentIndex < s.length-offset-1 && s.next() != nil {
			s2.next()
		} else {
			i = false
		}
	}
	return true
}

func (s *selIdentList) getSub(s2 *selIdentList) *selIdentList {
	if s2 == nil || s2.length > s.length {
		return nil
	}
	s.reset()
	s2.reset()
	for i := true; i; {
		if !s.current.isEqual(s2.current) {
			return nil
		}
		if s2.currentIndex != s2.length-2 { // might want to add a selNode.prev() func
			s.next()
			s2.next()
		} else {
			i = false
		}
	}
	return &selIdentList{
		start:        s.current,
		length:       s.length - s.currentIndex,
		current:      s.current,
		currentIndex: 0,
	}
}

func (s *selIdentList) changeRoot(r *ast.Ident, t types.Object) {
	selNode := &selIdentNode{
		this:   r,
		next:   s.start.next,
		typObj: t,
	}
	if s.start == s.current {
		s.start = selNode
		s.current = selNode
	} else {
		s.start = selNode
	}
}

func (s selIdentList) String() (str string) {
	var temp *selIdentNode = s.start
	str = fmt.Sprintf("length: %v\n[\n", s.length)
	for i := 0; temp != nil; i++ {
		if i == s.currentIndex {
			str += "*"
		}
		str += fmt.Sprintf("%v: %v\n", i, temp)
		temp = temp.next
	}
	str += "]"
	return str
}

func (s *selIdentNode) isEqual(s2 *selIdentNode) bool {
	return (s.this.Name == s2.this.Name) && (s.typObj == s2.typObj)
}

func (s selIdentNode) String() string {
	return fmt.Sprintf("{ ident: '%v', type: '%v' }", s.this, s.typObj)
}

func mapSelTypes(c *ast.CallExpr, pass *analysis.Pass) *selIdentList {
	list := &selIdentList{}
	valid := list.recurMapSelTypes(c.Fun, nil, pass.TypesInfo)
	if !valid {
		return nil
	}
	return list
}

func (l *selIdentList) recurMapSelTypes(e ast.Expr, next *selIdentNode, t *types.Info) bool {
	expr := astutil.Unparen(e)
	l.length++
	s := &selIdentNode{next: next}
	switch stmt := expr.(type) {
	case *ast.Ident:
		s.this = stmt
		s.typObj = t.ObjectOf(stmt)
	case *ast.SelectorExpr:
		s.this = stmt.Sel
		if sel, ok := t.Selections[stmt]; ok {
			s.typObj = sel.Obj() // method or field
		} else {
			s.typObj = t.Uses[stmt.Sel] // qualified identifier?
		}
		return l.recurMapSelTypes(stmt.X, s, t)
	default:
		return false
	}
	l.current = s
	l.start = s
	return true
}

type callInfo struct {
	call *ast.CallExpr
	id   string // type ID [either the name (if the function is exported) or the package/name if otherwise] of the function/method
	name string
}

// returns a *callInfo struct with call info (ID and type)
func getCallInfo(tInfo *types.Info, call *ast.CallExpr) (c *callInfo) {
	c = &callInfo{}
	c.call = call
	f := typeutil.Callee(tInfo, call)
	if f == nil {
		return nil
	}
	if _, isBuiltin := f.(*types.Builtin); isBuiltin {
		return nil
	}
	s, ok := f.Type().(*types.Signature)
	if ok && interfaceMethod(s) {
		return nil
	}
	c.id = f.String()
	c.name = f.Id()
	return c
}

func interfaceMethod(s *types.Signature) bool {
	recv := s.Recv()
	return recv != nil && types.IsInterface(recv.Type())
}

// hasNestedRLock takes a call expression represented by callInfo as input and returns a stack trace of the nested or recursive RLock within
// that call expression. If the call expression does not contain a nested or recursive RLock, hasNestedRLock returns an empty string.
// hasNestedRLock finds a nested or recursive RLock by recursively calling itself on any functions called by the function/method represented
// by callInfo.
func hasNestedRLock(fullRLockSelector *selIdentList, compareMap *selIdentList, call *callInfo, inspect *inspector.Inspector, pass *analysis.Pass, hist map[string]bool) (retStack string) {
	// debug := debugHelper{
	// 	pass: pass,
	// }
	var rLockSelector *selIdentList
	f := pass.Fset
	tInfo := pass.TypesInfo
	cH := callHelper{
		call: call.call,
		fset: pass.Fset,
	}
	var node ast.Node = cH.identifyFuncLitBlock(cH.call.Fun) // this seems a bit redundant
	var recv *ast.Ident
	if node == (*ast.BlockStmt)(nil) {
		subMap := fullRLockSelector.getSub(compareMap)
		if subMap != nil {
			rLockSelector = subMap
		} else {
			return "" // if this is not a local function literal call, and the selectors don't match up, then we can just return
		}
		node = findCallDeclarationNode(call, inspect, pass.TypesInfo)
		if node == (*ast.FuncDecl)(nil) {
			return ""
		} else if castedNode, ok := node.(*ast.FuncDecl); ok && castedNode.Recv != nil {
			recv = castedNode.Recv.List[0].Names[0]
			rLockSelector.changeRoot(recv, pass.TypesInfo.ObjectOf(recv))
		}
	} else {
		rLockSelector = fullRLockSelector // no need to find a submap, since this is a local function call
	}
	addition := fmt.Sprintf("\t%q at %v\n", call.name, f.Position(call.call.Pos()))
	ast.Inspect(node, func(iNode ast.Node) bool {
		switch stmt := iNode.(type) {
		case *ast.CallExpr:
			c := getCallInfo(tInfo, stmt)
			if c == nil {
				return false
			}
			name := c.name
			selMap := mapSelTypes(stmt, pass)
			if rLockSelector.isEqual(selMap, 0) { // if the method found is an RLock method
				retStack += addition + fmt.Sprintf("\t%q at %v\n", name, f.Position(iNode.Pos()))
			} else if name != "RUnlock" { // name should not equal the previousName to prevent infinite recursive loop
				nt := c.id
				if !hist[nt] { // make sure we are not in an infinite recursive loop
					hist[nt] = true
					stack := hasNestedRLock(rLockSelector, selMap, c, inspect, pass, hist)
					delete(hist, nt)
					if stack != "" {
						retStack += addition + stack
					}
				}
			}
		}
		return true
	})
	return retStack
}

// findCallDeclarationNode takes a callInfo struct and inspects the AST of the package
// to find a matching method or function declaration. It returns this declaration of type *ast.FuncDecl
func findCallDeclarationNode(c *callInfo, inspect *inspector.Inspector, tInfo *types.Info) *ast.FuncDecl {
	var retNode *ast.FuncDecl = nil
	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}
	inspect.Preorder(nodeFilter, func(node ast.Node) {
		funcDec, ok := node.(*ast.FuncDecl)
		if !ok {
			return
		}
		compareId := tInfo.ObjectOf(funcDec.Name).String()
		if c.id == compareId {
			retNode = funcDec
		}
	})
	return retNode
}

type callHelper struct {
	call *ast.CallExpr
	fset *token.FileSet
}

func (c callHelper) identifyFuncLitBlock(expr ast.Expr) *ast.BlockStmt {
	switch stmt := expr.(type) {
	case *ast.FuncLit:
		return stmt.Body
	case *ast.Ident:
		if stmt.Obj != nil {
			switch objDecl := stmt.Obj.Decl.(type) {
			case *ast.ValueSpec:
				identIndex := findIdentIndex(stmt, objDecl.Names)
				if identIndex != -1 && len(objDecl.Names) == len(objDecl.Values) {
					value := objDecl.Values[identIndex]
					return c.identifyFuncLitBlock(value)
				}
			case *ast.AssignStmt:
				exprIndex := findIdentIndexFromExpr(stmt, objDecl.Lhs)
				if exprIndex != -1 && len(objDecl.Lhs) == len(objDecl.Rhs) { // only deals with simple func lit assignments
					value := objDecl.Rhs[exprIndex]
					return c.identifyFuncLitBlock(value)
				}
			}
		}
	}
	return nil
}

func findIdentIndex(id *ast.Ident, exprs []*ast.Ident) int {
	for i, v := range exprs {
		if v.Name == id.Name {
			return i
		}
	}
	return -1
}

func findIdentIndexFromExpr(id *ast.Ident, exprs []ast.Expr) int {
	for i, v := range exprs {
		if val, ok := v.(*ast.Ident); ok && val.Name == id.Name {
			return i
		}
	}
	return -1
}
