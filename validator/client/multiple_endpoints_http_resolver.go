package client

import (
	"fmt"
	"net/http"

	"github.com/prysmaticlabs/prysm/v4/network/httputil"
)

// MultipleEndpointsHTTPResolver is a custom resolver for HTTP clients that supports multiple addresses.
type MultipleEndpointsHTTPResolver struct {
	addresses  []string
	currentIdx int
	client     *http.Client
}

// NewMultipleEndpointsHTTPResolver creates a new instance of MultipleEndpointsHTTPResolver.
func NewMultipleEndpointsHTTPResolver(addresses []string) *MultipleEndpointsHTTPResolver {
	return &MultipleEndpointsHTTPResolver{
		addresses:  addresses,
		currentIdx: 0,
		client:     &http.Client{},
	}
}

func (r *MultipleEndpointsHTTPResolver) HttpMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Attempt to send the request to the current endpoint
		err := r.sendRequest(req, w)

		// Switch to the next endpoint and retry if there is an error
		for i := 0; i < len(r.addresses)-1 && err != nil; i++ {
			r.switchEndpoint()
			err = r.sendRequest(req, w)
		}

		if err != nil {
			httputil.HandleError(w, fmt.Sprintf("failed to send request: %v", err), http.StatusInternalServerError)
			return
		}

		next.ServeHTTP(w, req)
	})
}

// sendRequest sends the HTTP request to the current endpoint.
func (r *MultipleEndpointsHTTPResolver) sendRequest(req *http.Request, w http.ResponseWriter) error {
	// Update the request URL with the current endpoint
	req.URL.Host = r.resolveEndpoint()

	// Send the HTTP request using the client
	resp, err := r.client.Do(req)
	if err != nil {
		// Optionally handle specific errors or log them
		httputil.HandleError(w, fmt.Sprintf("error sending request to %s: %v\n", r.resolveEndpoint(), err), http.StatusInternalServerError)
		return err
	}
	defer resp.Body.Close()
	return nil
}

// resolveEndpoint returns the current endpoint based on the resolver's state.
func (r *MultipleEndpointsHTTPResolver) resolveEndpoint() string {
	return r.addresses[r.currentIdx]
}

// switchToNextEndpoint switches to the next available endpoint, this is circular.
func (r *MultipleEndpointsHTTPResolver) switchEndpoint() {
	r.currentIdx = (r.currentIdx + 1) % len(r.addresses)
}
