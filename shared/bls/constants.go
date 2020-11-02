package bls

import (
	"github.com/prysmaticlabs/prysm/shared/bls/common"
)

// DomainByteLength length of domain byte array.
const DomainByteLength = 4

// CurveOrder for the BLS12-381 curve.
const CurveOrder = "52435875175126190479447740508185965837690552500527637822603658699938581184513"

// ZeroSecretKey represents a zero secret key.
var ZeroSecretKey = common.ZeroSecretKey

// InfinitePublicKey represents an infinite public key.
var InfinitePublicKey = common.InfinitePublicKey
