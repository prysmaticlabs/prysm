package testdata

func Type() {
	type error struct{}
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

func Parameter() {
	f := func(error int) { // want "Identifier 'error' shadows a predeclared identifier with the same name. Choose another name."
		if error == 0 {
		}
	}
	f(0)
}
