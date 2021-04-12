package powchain

import (
	"encoding/base64"
	"strings"

	"github.com/prysmaticlabs/prysm/shared/httputils"
	"github.com/prysmaticlabs/prysm/shared/httputils/authorizationmethod"
)

// HttpEndpoint extracts an httputils.Endpoint from the provider parameter.
func HttpEndpoint(eth1Provider string) httputils.Endpoint {
	endpoint := httputils.Endpoint{
		Endpoint: "",
		Auth: httputils.AuthorizationData{
			Method: authorizationmethod.None,
			Value:  "",
		}}

	authValues := strings.Split(eth1Provider, ",")
	if len(authValues) > 2 {
		log.Errorf(
			"ETH1 endpoint string can contain one comma for specifying the authorization header to access the provider."+
				" String contains too many commas: %d. Skipping authorization.", len(authValues)-1)
		endpoint.Endpoint = authValues[0]
	} else if len(authValues) == 2 {
		endpoint.Endpoint = authValues[0]
		switch httputils.Method(authValues[1]) {
		case authorizationmethod.Basic:
			basicAuthValues := strings.Split(authValues[1], " ")
			if len(basicAuthValues) != 2 {
				log.Errorf("Basic Authentication has incorrect format. Skipping authorization.")
			} else {
				endpoint.Auth.Method = authorizationmethod.Basic
				endpoint.Auth.Value = base64.StdEncoding.EncodeToString([]byte(basicAuthValues[1]))
			}
		case authorizationmethod.Bearer:
			endpoint.Auth.Method = authorizationmethod.Bearer
			endpoint.Auth.Value = authValues[1]
		case authorizationmethod.None:
			log.Errorf("Authorization has incorrect format or authorization type is not supported.")
		}
	} else if len(authValues) == 1 {
		endpoint.Endpoint = authValues[0]
	}
	return endpoint
}
