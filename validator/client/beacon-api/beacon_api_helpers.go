//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"math/big"
	"regexp"
	"strconv"

	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
)

func validRoot(root string) bool {
	matchesRegex, err := regexp.MatchString("^0x[a-fA-F0-9]{64}$", root)
	if err != nil {
		return false
	}
	return matchesRegex
}

func uint64ToString[T uint64 | types.Slot | types.ValidatorIndex | types.CommitteeIndex | types.Epoch](val T) string {
	return strconv.FormatUint(uint64(val), 10)
}

func littleEndianBytesToString(bytes []byte) string {
	// Integers are stored as little-endian, but big.Int expects big-endian. So we need to reverse the byte order before decoding.
	return new(big.Int).SetBytes(bytesutil.ReverseByteOrder(bytes)).String()
}
