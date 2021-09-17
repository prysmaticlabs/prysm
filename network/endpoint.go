package network

import (
	"errors"
	"strings"

	"github.com/prysmaticlabs/prysm/network/authorization"
)

// Endpoint is an endpoint with authorization data.
type Endpoint struct {
	Url  string
	Auth AuthorizationData
}

// AuthorizationData holds all information necessary to authorize with HTTP.
type AuthorizationData struct {
	Method authorization.AuthorizationMethod
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
	case authorization.Basic:
		return "Basic " + d.Value, nil
	case authorization.Bearer:
		return "Bearer " + d.Value, nil
	case authorization.None:
		return "", nil
	}

	return "", errors.New("could not create HTTP header for unknown authorization method")
}

// Method returns the authorizationmethod.AuthorizationMethod corresponding with the parameter value.
func Method(auth string) authorization.AuthorizationMethod {
	if strings.HasPrefix(strings.ToLower(auth), "basic") {
		return authorization.Basic
	}
	if strings.HasPrefix(strings.ToLower(auth), "bearer") {
		return authorization.Bearer
	}
	return authorization.None
}
