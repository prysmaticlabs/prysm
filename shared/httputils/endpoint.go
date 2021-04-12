package httputils

import (
	"errors"
	"strings"

	"github.com/prysmaticlabs/prysm/shared/httputils/authorizationmethod"
)

// Endpoint is an endpoint with authorization data.
type Endpoint struct {
	Url  string
	Auth AuthorizationData
}

// AuthorizationData holds all information necessary to authorize with HTTP.
type AuthorizationData struct {
	Method authorizationmethod.AuthorizationMethod
	Value  string
}

// ToHeaderValue retrieves the value of the authorization header from AuthorizationData.
func (e *AuthorizationData) ToHeaderValue() (string, error) {
	switch e.Method {
	case authorizationmethod.Basic:
		return "Basic " + e.Value, nil
	case authorizationmethod.Bearer:
		return "Bearer " + e.Value, nil
	case authorizationmethod.None:
		return "", nil
	}

	return "", errors.New("could not create HTTP header for unknown authorization method")
}

// Method returns the authorizationmethod.AuthorizationMethod corresponding with the parameter value.
func Method(auth string) authorizationmethod.AuthorizationMethod {
	if strings.HasPrefix(strings.ToLower(auth), "basic") {
		return authorizationmethod.Basic
	}
	if strings.HasPrefix(strings.ToLower(auth), "bearer") {
		return authorizationmethod.Bearer
	}
	return authorizationmethod.None
}
