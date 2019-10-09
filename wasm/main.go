package main

import (
	"fmt"

	"github.com/sirupsen/logrus"
	wasm "github.com/wasmerio/go-ext-wasm/wasmer"
)

type beaconState struct {
	Slot             uint64
	ExecutionScripts [][]byte
}

type shardState struct {
	Slot              uint64
	ExecEnvStateRoots [][32]byte
}

type shardBlock struct {
	Slot             uint64
	EnvironmentIndex uint64
	Data             []byte
}

func main() {
	// Reads the WebAssembly module as bytes.
	// TODO: Load multiple execution environment scripts in initialization.
	rawWasmCode, _ := wasm.ReadBytes("scripts/simple.wasm")
	bState := &beaconState{
		Slot:             0,
		ExecutionScripts: [][]byte{rawWasmCode},
	}
	sState := &shardState{
		Slot:              0,
		ExecEnvStateRoots: make([][32]byte, 1),
	}

	block := &shardBlock{
		Slot:             1,
		EnvironmentIndex: 0,
		Data:             []byte{1, 2, 3, 4, 5},
	}
	// Get the code from the beacon state exec env index.
	logrus.WithField(
		"slot",
		block.Slot,
	).Info("Processing shard block")
	code := bState.ExecutionScripts[block.EnvironmentIndex]
	shardPreStateRoot := sState.ExecEnvStateRoots[block.EnvironmentIndex]
	logrus.WithFields(logrus.Fields{
		"stateRoot":        fmt.Sprintf("%#x", shardPreStateRoot),
		"environmentIndex": block.EnvironmentIndex,
	}).Info("Running WASM code for shard block execution environment")
	shardPostStateRoot, err := ExecuteCode(code, shardPreStateRoot, block.Data)
	if err != nil {
		logrus.Fatal(err)
	}
	sState.ExecEnvStateRoots[block.EnvironmentIndex] = shardPostStateRoot
	logrus.WithField(
		"stateRoot",
		fmt.Sprintf("%#x", shardPostStateRoot),
	).Info("Updated shard state root")
}

func ExecuteCode(code []byte, preStateRoot [32]byte, shardData []byte) ([32]byte, error) {
	// Instantiates the WebAssembly module.
	instance, _ := wasm.NewInstance(code)
	defer instance.Close()
	sum := instance.Exports["sum"]

	// Calls that exported function with Go standard values. The WebAssembly
	// types are inferred and values are casted automatically.
	logrus.Infof("Executing sum function on %d and %d", 492, 3222)
	result, err := sum(492, 3222)
	if err != nil {
		return [32]byte{}, err
	}
	logrus.Infof("Code ran successfully - result = %s", result.String())
	return [32]byte{1}, nil
}
