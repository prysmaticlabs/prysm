package main

import (
	"fmt"

	wasm "github.com/wasmerio/go-ext-wasm/wasmer"
)

type shardBlockData struct {
}

type shardBlockHeader struct {
}

type shardState struct {
	Slot              uint64
	ParentBlock       *shardBlockHeader
	ExecEnvStateRoots [][32]byte
}

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

	// TODO: Get the code from the beacon state exec env index.
	// TODO: Get the EE pre-state root from the shard state exec env index.
	// TODO: Run code to get the post-state, update the state root at env index.
}

func ExecuteCode(code []byte, preStateRoot [32]byte, blockData *shardBlockData) ([32]byte, error) {
	return [32]byte{}, nil
}
