package testdata

func Struct() {
	type error struct{} // want "Type 'error' shadows a predeclared identifier with the same name. Choose another name."
}

func TypeAlias() {
	type error string // want "Type 'error' shadows a predeclared identifier with the same name. Choose another name."
}

func UninitializedVar() {
	var error int // want "Identifier 'error' shadows a predeclared identifier with the same name. Choose another name."
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

func Parameter() {
	f := func(error int) { // want "Identifier 'error' shadows a predeclared identifier with the same name. Choose another name."
		if error == 0 {
		}
	}
	f(0)
}
