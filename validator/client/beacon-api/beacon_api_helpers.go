//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"fmt"
	neturl "net/url"
	"regexp"
)

func validRoot(root string) bool {
	matchesRegex, err := regexp.MatchString("^0x[a-fA-F0-9]{64}$", root)
	if err != nil {
		return false
	}
	return matchesRegex
}

func buildURL(hostPort string, path string, queryParams ...neturl.Values) string {
	url := fmt.Sprintf("%s/%s", hostPort, path)

	if len(queryParams) == 0 {
		return url
	}

	return fmt.Sprintf("%s?%s", url, queryParams[0].Encode())
}
