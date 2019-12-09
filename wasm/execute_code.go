package wasm

import (
	"errors"

	"github.com/wasmerio/go-ext-wasm/wasmer"
)

//ExecuteCode executes wasm code in the ethereum environment
func executeCode(execCode []byte, preState [32]byte, blockData []byte) (postState [32]byte, deposits []Deposit, error error) {
	imports, err := getImports()
	if err != nil {
		log.WithError(err).Error("imports error")
		return preState, nil, err
	}

	instance, err := wasmer.NewInstanceWithImports(execCode, imports)
	if err != nil {
		log.WithError(err).Error("error creating instance")
		return preState, nil, err
	}
	defer instance.Close()

	main := instance.Exports["main"]
	if main == nil {
		log.Warnf("main function not exported. All exports: %v", instance.Exports)
		return preState, nil, errors.New("main function not exported")
	}
	result, err := main()

	if err != nil {
		log.WithError(err).Error("error executing instance")
		return preState, nil, err
	}

	log.WithField("result", result).Debug("executing instance")

	//TODO(#0): Get post_state_root and deposits
	return preState, nil, nil
}
