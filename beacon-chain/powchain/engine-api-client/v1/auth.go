package v1

import (
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/pkg/errors"
)

// Implements the http.RoundTripper interface to add JWT authentication
// support to an HTTP client used for interacting with an execution node.
// See the specification for more details on the supported JWT claims:
// https://github.com/ethereum/execution-apis.
type jwtTransport struct {
	underlyingTransport http.RoundTripper
	jwtSecret           []byte
}

// RoundTrip --
func (t *jwtTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iat": time.Now().Unix(), // Issued at.
	})
	tokenString, err := token.SignedString(t.jwtSecret)
	if err != nil {
		return nil, errors.Wrap(err, "could not produce signed JWT token")
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", tokenString))
	return t.underlyingTransport.RoundTrip(req)
}
