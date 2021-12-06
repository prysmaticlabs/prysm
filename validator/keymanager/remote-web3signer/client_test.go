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

func TestClient_Sign(t *testing.T) {
	json := `{ signature: "0xaj0dsfj0adsfj0asjdsjfjasdfk" }`
	// create a new reader with that JSON
	r := ioutil.NopCloser(bytes.NewReader([]byte(json)))
	GetDoFunc = func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       r,
		}, nil
	}

}

func TestClient_GetPublicKeys(t *testing.T) {
	json := `["example","example2"]`
	// create a new reader with that JSON
	r := ioutil.NopCloser(bytes.NewReader([]byte(json)))
	GetDoFunc = func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       r,
		}, nil
	}
	cl := client{BasePath: "example.com", restClient: &mockClient{}}
	resp, err := cl.GetPublicKeys()
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, "example", resp[0])
}

func TestClient_ReloadSignerKeys(t *testing.T) {
	GetDoFunc = func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewReader(nil)),
		}, nil
	}
	cl := client{BasePath: "example.com", restClient: &mockClient{}}
	err := cl.ReloadSignerKeys()
	assert.Nil(t, err)
}

func TestClient_GetServerStatus(t *testing.T) {
	json := `some server status, not sure what it looks like, need to find some sample data`
	r := ioutil.NopCloser(bytes.NewReader([]byte(json)))
	GetDoFunc = func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       r,
		}, nil
	}

}
