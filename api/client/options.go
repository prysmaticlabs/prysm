package client

import (
	"fmt"
	"net/http"
	"time"
)

type ReqOption func(*http.Request)

func WithSSZEncoding() ReqOption {
	return func(req *http.Request) {
		req.Header.Set("Accept", "application/octet-stream")
	}
}

func WithTokenAuthorization(token string) ReqOption {
	return func(req *http.Request) {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	}
}

// ClientOpt is a functional option for the Client type (http.Client wrapper)
type ClientOpt func(*Client)

// WithTimeout sets the .Timeout attribute of the wrapped http.Client.
func WithTimeout(timeout time.Duration) ClientOpt {
	return func(c *Client) {
		c.hc.Timeout = timeout
	}
}

// WithCustomTransport replaces the underlying http's transport with a custom one.
func WithCustomTransport(t http.RoundTripper) ClientOpt {
	return func(c *Client) {
		c.hc.Transport = t
	}
}

// WithTokenAuthentication sets an oauth token to be used.
func WithTokenAuthentication(token string) ClientOpt {
	return func(c *Client) {
		c.token = token
	}
}
