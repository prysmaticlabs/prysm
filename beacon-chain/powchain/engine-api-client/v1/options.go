package v1

import (
	"net/http"
)

// Option for configuring the engine API client.
type Option func(c *Client) error

type config struct {
	httpClient *http.Client
}

func defaultConfig() *config {
	return &config{
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
}

// WithJWTSecret allows setting a JWT secret for authenticating
// the client via HTTP connections.
func WithJWTSecret(secret []byte) Option {
	return func(c *Client) error {
		if len(secret) == 0 {
			return nil
		}
		authTransport := &jwtTransport{
			underlyingTransport: http.DefaultTransport,
			jwtSecret:           secret,
		}
		c.cfg.httpClient = &http.Client{
			Timeout:   DefaultTimeout,
			Transport: authTransport,
		}
		return nil
	}
}
