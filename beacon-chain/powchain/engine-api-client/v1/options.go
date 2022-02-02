package v1

import (
	"net/http"
	"time"
)

// Option for configuring the engine API client.
type Option func(c *Client) error

type config struct {
	httpClient *http.Client
}

func defaultConfig() *config {
	return &config{
		httpClient: &http.Client{
			Timeout: time.Second * 5,
		},
	}
}

// WithHTTPClient allows setting a custom HTTP client
// for the API connection.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) error {
		c.cfg.httpClient = httpClient
		return nil
	}
}
