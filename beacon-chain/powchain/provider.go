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
		Url: "",
		Auth: httputils.AuthorizationData{
			Method: authorizationmethod.None,
			Value:  "",
		}}

	authValues := strings.Split(eth1Provider, ",")
	endpoint.Url = strings.TrimSpace(authValues[0])
	if len(authValues) > 2 {
		log.Errorf(
			"ETH1 endpoint string can contain one comma for specifying the authorization header to access the provider."+
				" String contains too many commas: %d. Skipping authorization.", len(authValues)-1)
	} else if len(authValues) == 2 {
		switch httputils.Method(strings.TrimSpace(authValues[1])) {
		case authorizationmethod.Basic:
			basicAuthValues := strings.Split(strings.TrimSpace(authValues[1]), " ")
			if len(basicAuthValues) != 2 {
				log.Errorf("Basic Authentication has incorrect format. Skipping authorization.")
			} else {
				endpoint.Auth.Method = authorizationmethod.Basic
				endpoint.Auth.Value = base64.StdEncoding.EncodeToString([]byte(basicAuthValues[1]))
			}
		case authorizationmethod.Bearer:
			bearerAuthValues := strings.Split(strings.TrimSpace(authValues[1]), " ")
			if len(bearerAuthValues) != 2 {
				log.Errorf("Bearer Authentication has incorrect format. Skipping authorization.")
			} else {
				endpoint.Auth.Method = authorizationmethod.Bearer
				endpoint.Auth.Value = bearerAuthValues[1]
			}
		case authorizationmethod.None:
			log.Errorf("Authorization has incorrect format or authorization type is not supported.")
		}
	}
	return endpoint
}
