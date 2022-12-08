package beacon_api

import (
	"fmt"
	neturl "net/url"
	"regexp"
	"strconv"

	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
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

func buildURL(path string, queryParams ...neturl.Values) string {
	if len(queryParams) == 0 {
		return path
	}

	return fmt.Sprintf("%s?%s", path, queryParams[0].Encode())
}
