package remote_web3signer

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

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

func TestClient_Sign_HappyPath(t *testing.T) {
	json := `{
  		"signature": "0xb3baa751d0a9132cfe93e4e3d5ff9075111100e3789dca219ade5a24d27e19d16b3353149da1833e9b691bb38634e8dc04469be7032132906c927d7e1a49b414730612877bc6b2810c8f202daf793d1ab0d6b5cb21d52f9e52e883859887a5d9"
	}`
	// create a new reader with that JSON
	r := ioutil.NopCloser(bytes.NewReader([]byte(json)))
	mock := &mockTransport{mockResponse: &http.Response{
		StatusCode: 200,
		Body:       r,
	}}
	cl := client{BasePath: "example.com", restClient: &http.Client{Transport: mock}}
	forkData := &Fork{
		PreviousVersion: "",
		CurrentVersion:  "",
		Epoch:           "",
	}
	forkInfoData := &ForkInfo{
		Fork:                  forkData,
		GenesisValidatorsRoot: "",
	}

	AggregationSlotData := &AggregationSlot{Slot: ""}
	// remember to replace signing root with hex encoding remove 0x
	web3SignerRequest := SignRequest{
		Type:            "foo",
		ForkInfo:        forkInfoData,
		SigningRoot:     "0xfasd0fjsa0dfjas0dfjasdf",
		AggregationSlot: AggregationSlotData,
	}
	resp, err := cl.Sign("a2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820", &web3SignerRequest)
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, "0xb3baa751d0a9132cfe93e4e3d5ff9075111100e3789dca219ade5a24d27e19d16b3353149da1833e9b691bb38634e8dc04469be7032132906c927d7e1a49b414730612877bc6b2810c8f202daf793d1ab0d6b5cb21d52f9e52e883859887a5d9", fmt.Sprintf("%#x", resp.Marshal()))
}

func TestClient_GetPublicKeys_HappyPath(t *testing.T) {
	// public keys are returned hex encoded with 0x
	json := `["0x613262356161616439633665666566653762623962313234336130343334303466333336323933376366623662333138333339323938333331373366343736363330656132636665623064396464663135663937636138363835393438383230","0x613262356161616439633665666566653762623962313234336130343334303466333336323933376366623662333138333339323938333331373366343736363330656132636665623064396464663135663937636138363835393438383230"]`
	// create a new reader with that JSON
	r := ioutil.NopCloser(bytes.NewReader([]byte(json)))
	mock := &mockTransport{mockResponse: &http.Response{
		StatusCode: 200,
		Body:       r,
	}}
	cl := client{BasePath: "example.com", restClient: &http.Client{Transport: mock}}
	resp, err := cl.GetPublicKeys()
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	// we would like them as 48byte base64 without 0x
	assert.EqualValues(t, "a2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820", string(resp[0]))
}

func TestClient_ReloadSignerKeys_HappyPath(t *testing.T) {
	mock := &mockTransport{mockResponse: &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewReader(nil)),
	}}
	cl := client{BasePath: "example.com", restClient: &http.Client{Transport: mock}}
	err := cl.ReloadSignerKeys()
	assert.Nil(t, err)
}

func TestClient_GetServerStatus_HappyPath(t *testing.T) {
	json := `"some server status, not sure what it looks like, need to find some sample data"`
	r := ioutil.NopCloser(bytes.NewReader([]byte(json)))
	mock := &mockTransport{mockResponse: &http.Response{
		StatusCode: 200,
		Body:       r,
	}}
	cl := client{BasePath: "example.com", restClient: &http.Client{Transport: mock}}
	resp, err := cl.GetServerStatus()
	assert.NotNil(t, resp)
	assert.Nil(t, err)
}
