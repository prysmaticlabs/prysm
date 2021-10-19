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

// Equals compares two endpoints for equality.
func (e Endpoint) Equals(other Endpoint) bool {
	return e.Url == other.Url && e.Auth.Equals(other.Auth)
}

// Equals compares two authorization data objects for equality.
func (d AuthorizationData) Equals(other AuthorizationData) bool {
	return d.Method == other.Method && d.Value == other.Value
}

// ToHeaderValue retrieves the value of the authorization header from AuthorizationData.
func (d *AuthorizationData) ToHeaderValue() (string, error) {
	switch d.Method {
	case authorizationmethod.Basic:
		return "Basic " + d.Value, nil
	case authorizationmethod.Bearer:
		return "Bearer " + d.Value, nil
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
