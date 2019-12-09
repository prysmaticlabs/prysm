package wasm

// #include <stdint.h>
//
// extern void eth2_loadPreStateRoot(void *context, int32_t x);
// extern int32_t eth2_blockDataSize(void *context);
// extern void eth2_savePostStateRoot(void *context, int32_t x);
import "C"

//interface determined in https://github.com/ewasm/ewasm-rust-api/blob/master/src/eth2.rs
//pub fn eth2_loadPreStateRoot(offset: *const u32);
//pub fn eth2_blockDataSize() -> u32;
//pub fn eth2_blockDataCopy(outputOfset: *const u32, offset: u32, length: u32);
//pub fn eth2_savePostStateRoot(offset: *const u32);
//pub fn eth2_pushNewDeposit(offset: *const u32, length: u32);

import (
	"unsafe"
	"github.com/wasmerio/go-ext-wasm/wasmer"
)

//TODO(#0): determine eth2_blockDataCopy
//TODO(#0): determine eth2_pushNewDeposit

//export eth2_loadPreStateRoot
func eth2_loadPreStateRoot(context unsafe.Pointer, arg int32) {
	//TODO(#0): Get "memory := instance.Memory.Data()".  Use context for this? Multithreading?
	uint32conversion := uint32(arg) // 1048544. This value realy exist in wasm
	log.WithField("arg", uint32conversion).Debug("eth2_loadPreStateRoot")
	return
}

//export eth2_blockDataSize
func eth2_blockDataSize(context unsafe.Pointer) int32 {
	log.Debug("eth2_blockDataSize")
	return 0
}

//export eth2_savePostStateRoot
func eth2_savePostStateRoot(context unsafe.Pointer, arg int32) {
	uint32conversion := uint32(arg) // 1048544. This value realy exist in wasm
	log.WithField("arg", uint32conversion).Debug("eth2_loadPreStateRoot")
}

func getImports() (*wasmer.Imports, error) {
	imports := wasmer.NewImports().Namespace("env")
	//TODO(#0): append memory
	imports, err := imports.Append("eth2_loadPreStateRoot", eth2_loadPreStateRoot, C.eth2_loadPreStateRoot)
	if err != nil {
		return nil, err
	}
	imports, err = imports.Append("eth2_blockDataSize", eth2_blockDataSize, C.eth2_blockDataSize)
	if err != nil {
		return nil, err
	}
	imports, err = imports.Append("eth2_savePostStateRoot", eth2_savePostStateRoot, C.eth2_savePostStateRoot)
	if err != nil {
		return nil, err
	}
	return imports, nil
}
