package main

import (
	"fmt"
	"log"

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
	Slot         uint64
	Transactions []*transaction
}

type transaction struct {
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
		Slot: 1,
		Transactions: []*transaction{
			{
				EnvironmentIndex: 0,
				Data:             []byte{1, 2, 3, 4, 5},
			},
			{
				EnvironmentIndex: 0,
				Data:             []byte{1, 2, 3, 4, 5},
			},
			{
				EnvironmentIndex: 0,
				Data:             []byte{1, 2, 3, 4, 5},
			},
		},
	}
	// Get the code from the beacon state exec env index.
	logrus.WithField(
		"slot",
		block.Slot,
	).Info("Processing shard block")
	if _, err := processShardBlock(bState, sState, block); err != nil {
		log.Fatal(err)
	}
}

func processShardBlock(bState *beaconState, sState *shardState, block *shardBlock) (*shardState, error) {
	for i := 0; i < len(block.Transactions); i++ {
		tx := block.Transactions[i]
		code := bState.ExecutionScripts[tx.EnvironmentIndex]
		shardPreStateRoot := sState.ExecEnvStateRoots[tx.EnvironmentIndex]
		logrus.WithFields(logrus.Fields{
			"stateRoot":        fmt.Sprintf("%#x", shardPreStateRoot),
			"environmentIndex": tx.EnvironmentIndex,
			"transactionID":    i,
		}).Info("Running WASM code for shard block transaction")
		shardPostStateRoot, err := executeCode(code, shardPreStateRoot, tx.Data)
		if err != nil {
			return nil, err
		}
		sState.ExecEnvStateRoots[tx.EnvironmentIndex] = shardPostStateRoot
		logrus.WithFields(logrus.Fields{
			"stateRoot":        fmt.Sprintf("%#x", shardPostStateRoot),
			"environmentIndex": tx.EnvironmentIndex,
		}).Info("Updated shard state root for environment index")
	}
	return sState, nil
}

func executeCode(code []byte, preStateRoot [32]byte, shardData []byte) ([32]byte, error) {
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
