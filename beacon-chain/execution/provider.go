package execution

import (
	"encoding/base64"
	"strings"

	"github.com/prysmaticlabs/prysm/v3/network"
	"github.com/prysmaticlabs/prysm/v3/network/authorization"
)

// HttpEndpoint extracts an httputils.Endpoint from the provider parameter.
func HttpEndpoint(eth1Provider string) network.Endpoint {
	endpoint := network.Endpoint{
		Url: "",
		Auth: network.AuthorizationData{
			Method: authorization.None,
			Value:  "",
		}}

	authValues := strings.Split(eth1Provider, ",")
	endpoint.Url = strings.TrimSpace(authValues[0])
	if len(authValues) > 2 {
		log.Errorf(
			"ETH1 endpoint string can contain one comma for specifying the authorization header to access the provider."+
				" String contains too many commas: %d. Skipping authorization.", len(authValues)-1)
	} else if len(authValues) == 2 {
		switch network.Method(strings.TrimSpace(authValues[1])) {
		case authorization.Basic:
			basicAuthValues := strings.Split(strings.TrimSpace(authValues[1]), " ")
			if len(basicAuthValues) != 2 {
				log.Errorf("Basic Authentication has incorrect format. Skipping authorization.")
			} else {
				endpoint.Auth.Method = authorization.Basic
				endpoint.Auth.Value = base64.StdEncoding.EncodeToString([]byte(basicAuthValues[1]))
			}
		case authorization.Bearer:
			bearerAuthValues := strings.Split(strings.TrimSpace(authValues[1]), " ")
			if len(bearerAuthValues) != 2 {
				log.Errorf("Bearer Authentication has incorrect format. Skipping authorization.")
			} else {
				endpoint.Auth.Method = authorization.Bearer
				endpoint.Auth.Value = bearerAuthValues[1]
			}
		case authorization.None:
			log.Errorf("Authorization has incorrect format or authorization type is not supported.")
		}
	}
	return endpoint
}
