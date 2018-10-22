package bls

/*
#cgo LDFLAGS: -lstdc++
#cgo CFLAGS:-DMCLBN_FP_UNIT_SIZE=6
#include "external/herumi_mcl/include/mcl/bn.h"
*/
import "C"

// CurveFp254BNb -- 254 bit curve
const CurveFp254BNb = C.mclBn_CurveFp254BNb

// CurveFp382_1 -- 382 bit curve 1
const CurveFp382_1 = C.mclBn_CurveFp382_1

// CurveFp382_2 -- 382 bit curve 2
const CurveFp382_2 = C.mclBn_CurveFp382_2

// BLS12381 -- 381 curve spec.
const BLS12381 = C.MCL_BLS12_381

func getBLS12381Curve() int {
	return BLS12381
}
