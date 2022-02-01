package internal_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"

	"github.com/prysmaticlabs/prysm/validator/keymanager/remote-web3signer/internal"
	"github.com/stretchr/testify/assert"
)

// mockTransport is the mock Transport object
type mockTransport struct {
	mockResponse *http.Response
}

// RoundTrip is mocking my own implementation of the RoundTripper interface
func (m *mockTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return m.mockResponse, nil
}

func TestNewApiClient(t *testing.T) {
	apiClient, err := internal.NewApiClient("http://localhost:8545")
	assert.NoError(t, err)
	assert.NotNil(t, apiClient)
}

func TestClient_Sign_HappyPath(t *testing.T) {
	jsonSig := `0xb3baa751d0a9132cfe93e4e3d5ff9075111100e3789dca219ade5a24d27e19d16b3353149da1833e9b691bb38634e8dc04469be7032132906c927d7e1a49b414730612877bc6b2810c8f202daf793d1ab0d6b5cb21d52f9e52e883859887a5d9`
	// create a new reader with that JSON
	r := ioutil.NopCloser(bytes.NewReader([]byte(jsonSig)))
	mock := &mockTransport{mockResponse: &http.Response{
		StatusCode: 200,
		Body:       r,
	}}
	u, err := url.Parse("example.com")
	assert.NoError(t, err)
	cl := internal.ApiClient{BaseURL: u, RestClient: &http.Client{Transport: mock}}
	jsonRequest, err := json.Marshal(`{message: "hello"}`)
	assert.NoError(t, err)
	resp, err := cl.Sign(context.Background(), "a2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820", jsonRequest)
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, "0xb3baa751d0a9132cfe93e4e3d5ff9075111100e3789dca219ade5a24d27e19d16b3353149da1833e9b691bb38634e8dc04469be7032132906c927d7e1a49b414730612877bc6b2810c8f202daf793d1ab0d6b5cb21d52f9e52e883859887a5d9", fmt.Sprintf("%#x", resp.Marshal()))
}

func TestClient_Sign_500(t *testing.T) {
	jsonSig := `0xb3baa751d0a9132cfe93e4e3d5ff9075111100e3789dca219ade5a24d27e19d16b3353149da1833e9b691bb38634e8dc04469be7032132906c927d7e1a49b414730612877bc6b2810c8f202daf793d1ab0d6b5cb21d52f9e52e883859887a5d9`
	// create a new reader with that JSON
	r := ioutil.NopCloser(bytes.NewReader([]byte(jsonSig)))
	mock := &mockTransport{mockResponse: &http.Response{
		StatusCode: 500,
		Body:       r,
	}}
	u, err := url.Parse("example.com")
	assert.NoError(t, err)
	cl := internal.ApiClient{BaseURL: u, RestClient: &http.Client{Transport: mock}}
	jsonRequest, err := json.Marshal(`{message: "hello"}`)
	assert.NoError(t, err)
	resp, err := cl.Sign(context.Background(), "a2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820", jsonRequest)
	assert.NotNil(t, err)
	assert.Nil(t, resp)

}

func TestClient_Sign_412(t *testing.T) {
	jsonSig := `0xb3baa751d0a9132cfe93e4e3d5ff9075111100e3789dca219ade5a24d27e19d16b3353149da1833e9b691bb38634e8dc04469be7032132906c927d7e1a49b414730612877bc6b2810c8f202daf793d1ab0d6b5cb21d52f9e52e883859887a5d9`
	// create a new reader with that JSON
	r := ioutil.NopCloser(bytes.NewReader([]byte(jsonSig)))
	mock := &mockTransport{mockResponse: &http.Response{
		StatusCode: 412,
		Body:       r,
	}}
	u, err := url.Parse("example.com")
	assert.NoError(t, err)
	cl := internal.ApiClient{BaseURL: u, RestClient: &http.Client{Transport: mock}}
	jsonRequest, err := json.Marshal(`{message: "hello"}`)
	assert.NoError(t, err)
	resp, err := cl.Sign(context.Background(), "a2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820", jsonRequest)
	assert.NotNil(t, err)
	assert.Nil(t, resp)

}

func TestClient_Sign_400(t *testing.T) {
	jsonSig := `0xb3baa751d0a9132cfe93e4e3d5ff9075111100e3789dca219ade5a24d27e19d16b3353149da1833e9b691bb38634e8dc04469be7032132906c927d7e1a49b414730612877bc6b2810c8f202daf793d1ab0d6b5cb21d52f9e52e883859887a5d9`
	// create a new reader with that JSON
	r := ioutil.NopCloser(bytes.NewReader([]byte(jsonSig)))
	mock := &mockTransport{mockResponse: &http.Response{
		StatusCode: 400,
		Body:       r,
	}}
	u, err := url.Parse("example.com")
	assert.NoError(t, err)
	cl := internal.ApiClient{BaseURL: u, RestClient: &http.Client{Transport: mock}}
	jsonRequest, err := json.Marshal(`{message: "hello"}`)
	assert.NoError(t, err)
	resp, err := cl.Sign(context.Background(), "a2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820", jsonRequest)
	assert.NotNil(t, err)
	assert.Nil(t, resp)

}

func TestClient_GetPublicKeys_HappyPath(t *testing.T) {
	// public keys are returned hex encoded with 0x
	json := `["0xa2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820"]`
	// create a new reader with that JSON
	r := ioutil.NopCloser(bytes.NewReader([]byte(json)))
	mock := &mockTransport{mockResponse: &http.Response{
		StatusCode: 200,
		Body:       r,
	}}
	u, err := url.Parse("example.com")
	assert.NoError(t, err)
	cl := internal.ApiClient{BaseURL: u, RestClient: &http.Client{Transport: mock}}
	resp, err := cl.GetPublicKeys(context.Background(), "example.com/api/publickeys")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	// we would like them as 48byte base64 without 0x
	assert.EqualValues(t, "[162 181 170 173 156 110 254 254 123 185 177 36 58 4 52 4 243 54 41 55 207 182 179 24 51 146 152 51 23 63 71 102 48 234 44 254 176 217 221 241 95 151 202 134 133 148 136 32]", fmt.Sprintf("%v", resp[0][:]))
}

func TestClient_GetPublicKeys_EncodingError(t *testing.T) {
	// public keys are returned hex encoded with 0x
	json := `["a2b5aaad9c6efefe7bb9b1243a043404f3362937c","fb6b31833929833173f476630ea2cfe","b0d9ddf15fca8685948820"]`
	// create a new reader with that JSON
	r := ioutil.NopCloser(bytes.NewReader([]byte(json)))
	mock := &mockTransport{mockResponse: &http.Response{
		StatusCode: 200,
		Body:       r,
	}}
	u, err := url.Parse("example.com")
	assert.NoError(t, err)
	cl := internal.ApiClient{BaseURL: u, RestClient: &http.Client{Transport: mock}}
	resp, err := cl.GetPublicKeys(context.Background(), "example.com/api/publickeys")
	assert.Equal(t, err.Error(), "failed to decode from Hex from the following public key index locations: 0, 1, 2, ")
	assert.Nil(t, resp)
}

// TODO: not really in use, should be revisited
func TestClient_ReloadSignerKeys_HappyPath(t *testing.T) {
	mock := &mockTransport{mockResponse: &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewReader(nil)),
	}}
	u, err := url.Parse("example.com")
	assert.NoError(t, err)
	cl := internal.ApiClient{BaseURL: u, RestClient: &http.Client{Transport: mock}}
	err = cl.ReloadSignerKeys(context.Background())
	assert.Nil(t, err)
}

// TODO: not really in use, should be revisited
func TestClient_GetServerStatus_HappyPath(t *testing.T) {
	json := `"some server status, not sure what it looks like, need to find some sample data"`
	r := ioutil.NopCloser(bytes.NewReader([]byte(json)))
	mock := &mockTransport{mockResponse: &http.Response{
		StatusCode: 200,
		Body:       r,
	}}
	u, err := url.Parse("example.com")
	assert.NoError(t, err)
	cl := internal.ApiClient{BaseURL: u, RestClient: &http.Client{Transport: mock}}
	resp, err := cl.GetServerStatus(context.Background())
	assert.NotNil(t, resp)
	assert.Nil(t, err)
}
