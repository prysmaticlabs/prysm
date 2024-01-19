package api

import "strings"

const WebUrlPrefix = "/v2/validator/"

func IsKeymanagerUrlPrefix(path string) bool {
	if strings.Contains(path, "/eth/v1/keystores") || strings.Contains(path, "/eth/v1/remotekeys") || strings.Contains(path, "/eth/v1/validator") {
		return true
	}
	return false
}
