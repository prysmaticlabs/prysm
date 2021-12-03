package remote_web3signer

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"testing"
)

// MockClient is the mock client
type mockClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

var (
	// GetDoFunc fetches the mock client's `Do` func
	GetDoFunc func(req *http.Request) (*http.Response, error)
)

// Do is the mock client's `Do` func
func (m *mockClient) Do(req *http.Request) (*http.Response, error) {
	return GetDoFunc(req)
}

func Test(t *testing.T) {
	json := `["example","example2"]`
	// create a new reader with that JSON
	r := ioutil.NopCloser(bytes.NewReader([]byte(json)))
	GetDoFunc = func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       r,
		}, nil
	}
	cl := client{BasePath: "example.com", APIPath: "/api/v1/eth", restClient: &mockClient{}}
	resp, err := cl.GetPublicKeys()
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, "example", resp[0])
}
