package main

import (
	"fmt"

	wasm "github.com/wasmerio/go-ext-wasm/wasmer"
)

func main() {
	// Reads the WebAssembly module as bytes.
	bytes, _ := wasm.ReadBytes("scripts/simple.wasm")

	// Instantiates the WebAssembly module.
	instance, _ := wasm.NewInstance(bytes)
	defer instance.Close()

	sum := instance.Exports["sum"]

	// Calls that exported function with Go standard values. The WebAssembly
	// types are inferred and values are casted automatically.
	result, _ := sum(492, 3222)
	fmt.Println(result)
}
