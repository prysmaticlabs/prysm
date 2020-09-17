package testdata

type foo struct {
}

func AddressOfDereferencedValue() {
	x := &foo{}
	_ = &*x // want "Found a no-op instruction that can be safely removed. It might be a result of writing code that does not work as expected."
}

func DereferencedAddressOfValue() {
	x := foo{}
	_ = *&x // want "Found a no-op instruction that can be safely removed. It might be a result of writing code that does not work as expected."
}
