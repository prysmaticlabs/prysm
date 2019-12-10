package wasm

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/wasmerio/go-ext-wasm/wasmer"
)

type beaconState struct {
	Slot             uint64
	ExecutionScripts [][]byte
}

//ShardState describes struct for the state transition function
//TODO(#0): Have other describes from buterin: https://notes.ethereum.org/@vbuterin/Bkoaj4xpN?type=view#Shard-processing
type ShardState struct {
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

//Deposit can be returned by the executeCode metod
//TODO(#0): use type from main beacon-chain code. For example Deposit from proto/eth/v1alpha1/beacon_block.pb.go
type Deposit struct {
	PubKey                [48]byte
	WithdrawalCredentials [48]byte
	Amount                uint64
}

var log = logrus.WithField("prefix", "wasm")

//TODO(#0): move to _test file
func main() {
	// Reads the WebAssembly module as bytes.
	// TODO(#0): Load multiple execution environment scripts in initialization.
	rawWasmCode, _ := wasmer.ReadBytes("tests/wasm.wasm")
	bState := &beaconState{
		Slot:             0,
		ExecutionScripts: [][]byte{rawWasmCode},
	}
	sState := &ShardState{
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
	if _, err := ProcessShardBlock(bState, sState, block); err != nil {
		log.Fatal(err)
	}
}

//ProcessShardBlock is the state transition function
func ProcessShardBlock(bState *beaconState, sState *ShardState, block *shardBlock) (*ShardState, error) {
	for i := 0; i < len(block.Transactions); i++ {
		tx := block.Transactions[i]
		code := bState.ExecutionScripts[tx.EnvironmentIndex]
		shardPreStateRoot := sState.ExecEnvStateRoots[tx.EnvironmentIndex]
		logrus.WithFields(logrus.Fields{
			"stateRoot":        fmt.Sprintf("%#x", shardPreStateRoot),
			"environmentIndex": tx.EnvironmentIndex,
			"transactionID":    i,
		}).Info("Running WASM code for shard block transaction")
		//TODO(#0): receive and process deposits from executeCode
		shardPostStateRoot, _ /*deposits*/, err := executeCode(code, shardPreStateRoot, tx.Data)
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
