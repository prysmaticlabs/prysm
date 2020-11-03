package bls

import (
	"github.com/prysmaticlabs/prysm/shared/bls/common"
)

// ErrZeroKey describes an error due to a zero secret key.
var ErrZeroKey = common.ErrZeroKey

// ErrInfinitePubKey describes an error due to an infinite public key.
var ErrInfinitePubKey = common.ErrInfinitePubKey
