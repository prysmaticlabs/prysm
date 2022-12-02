//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"regexp"
)

func validRoot(root string) bool {
	matchesRegex, err := regexp.MatchString("^0x[a-fA-F0-9]{64}$", root)
	if err != nil {
		return false
	}
	return matchesRegex
}
