package testdata

type len struct { // want "Type 'len' shadows a predeclared identifier with the same name. Choose another name."

}

type int interface { // want "Type 'int' shadows a predeclared identifier with the same name. Choose another name."

}

func Struct() {
	type error struct { // want "Type 'error' shadows a predeclared identifier with the same name. Choose another name."
		int int // No diagnostic because the name is always referenced indirectly through a struct variable.
	}
}

func TypeAlias() {
	type error string // want "Type 'error' shadows a predeclared identifier with the same name. Choose another name."
}

func UninitializedVarAndAssignments() {
	var error int // want "Identifier 'error' shadows a predeclared identifier with the same name. Choose another name."
	error = 1     // No diagnostic because the original declaration already triggered one.
	if error == 0 {
	}
}

func InitializedVar() {
	error := 0 // want "Identifier 'error' shadows a predeclared identifier with the same name. Choose another name."
	if error == 0 {
	}
}

func FirstInVarList() {
	error, x := 0, 1 // want "Identifier 'error' shadows a predeclared identifier with the same name. Choose another name."
	if error == x {
	}
}

func SecondInVarList() {
	x, error := 0, 1 // want "Identifier 'error' shadows a predeclared identifier with the same name. Choose another name."
	if error == x {
	}
}

func Const() {
	const error = 0 // want "Identifier 'error' shadows a predeclared identifier with the same name. Choose another name."
}

// Test function and parameter names.
func error(len int) { // want "Function 'error' shadows a predeclared identifier with the same name. Choose another name." "Identifier 'len' shadows a predeclared identifier with the same name. Choose another name."
	if len == 0 {
	}

	// Test parameter in a new line.
	f := func(
		int string) { // want "Identifier 'int' shadows a predeclared identifier with the same name. Choose another name."
	}

	f("")
}

type receiver struct {
}

// Test receiver function.
func (s *receiver) Receiver(len int) {

}
