package remote_web3signer

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/crypto/bls"
)

const (
	ethApiNamespace = "/api/v1/eth2/sign/"
	maxTimeout      = 3 * time.Second
)

// client a wrapper object around web3signer APIs. API docs found here https://consensys.github.io/web3signer/web3signer-eth2.html.
type client struct {
	BasePath   string
	restClient *http.Client
}

// newClient method instantiates a new client object.
//nolint:unused,deadcode
func newClient(endpoint string) (*client, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "invalid format, unable to parse url")
	}
	return &client{
		BasePath: u.Host,
		restClient: &http.Client{
			Timeout: maxTimeout,
		},
	}, nil
}

// SignRequest is a request object for web3signer sign api.
type SignRequest struct {
	Type            string           `json:"type"`
	ForkInfo        *ForkInfo        `json:"fork_info"`
	SigningRoot     string           `json:"signingRoot"`
	AggregationSlot *AggregationSlot `json:"aggregation_slot"`
}

// ForkInfo a sub property object of the Sign request,in the future before the merge to remove the need to send the entire block body and just use the block_body_root.
type ForkInfo struct {
	Fork                  *Fork  `json:"fork"`
	GenesisValidatorsRoot string `json:"genesis_validators_root"`
}

// Fork a sub property of ForkInfo.
type Fork struct {
	PreviousVersion string `json:"previous_version"`
	CurrentVersion  string `json:"current_version"`
	Epoch           string `json:"epoch"`
}

// AggregationSlot a sub property of SignRequest.
type AggregationSlot struct {
	Slot string `json:"slot"`
}

// signResponse the response object of the web3signer sign api.
type signResponse struct {
	Signature string `json:"signature"`
}

// Sign is a wrapper method around the web3signer sign api.
func (client *client) Sign(pubKey string, request *SignRequest) (bls.Signature, error) {
	requestPath := ethApiNamespace + pubKey
	jsonRequest, err := json.Marshal(request)
	if err != nil {
		return nil, errors.Wrap(err, "invalid format, failed to marshal json request")
	}
	resp, err := client.doRequest(http.MethodPost, client.BasePath+requestPath, bytes.NewBuffer(jsonRequest))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == 404 {
		return nil, errors.Wrap(err, "public key not found")
	}
	if resp.StatusCode == 412 {
		return nil, errors.Wrap(err, "signing operation failed due to slashing protection rules")
	}
	signResp := &signResponse{}
	if err := client.unmarshalResponse(resp.Body, &signResp); err != nil {
		return nil, err
	}
	decoded, err := decodeHex(signResp.Signature)
	if err != nil {
		return nil, err
	}
	return bls.SignatureFromBytes(decoded)
}

// GetPublicKeys is a wrapper method around the web3signer publickeys api (this may be removed in the future or moved to another location due to its usage).
func (client *client) GetPublicKeys() ([][]byte, error) {
	const requestPath = "/publicKeys"
	resp, err := client.doRequest(http.MethodGet, client.BasePath+requestPath, nil)
	if err != nil {
		return nil, err
	}
	var publicKeys []string
	if err := client.unmarshalResponse(resp.Body, &publicKeys); err != nil {
		return nil, err
	}
	decodedKeys := make([][]byte, len(publicKeys))
	var errorKeyPositions string
	for i, value := range publicKeys {
		decodedKey, err := decodeHex(value)
		if err != nil {
			errorKeyPositions += fmt.Sprintf("%v, ", i)
			continue
		}
		decodedKeys[i] = decodedKey
	}
	if errorKeyPositions != "" {
		return nil, errors.New("failed to decode from Hex from the following public key index locations: " + errorKeyPositions)
	}
	return decodedKeys, nil
}

// ReloadSignerKeys is a wrapper method around the web3signer reload api.
func (client *client) ReloadSignerKeys() error {
	const requestPath = "/reload"
	if _, err := client.doRequest(http.MethodPost, client.BasePath+requestPath, nil); err != nil {
		return err
	}
	return nil
}

// GetServerStatus is a wrapper method around the web3signer upcheck api
func (client *client) GetServerStatus() (string, error) {
	const requestPath = "/upcheck"
	resp, err := client.doRequest(http.MethodGet, client.BasePath+requestPath, nil)
	if err != nil {
		return "", err
	}
	var status string
	if err := client.unmarshalResponse(resp.Body, &status); err != nil {
		return "", err
	}
	return status, nil
}

// doRequest is a utility method for requests.
func (client *client) doRequest(httpMethod, fullPath string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(httpMethod, fullPath, body)
	if err != nil {
		return nil, errors.Wrap(err, "invalid format, failed to create new Post Request Object")
	}
	resp, err := client.restClient.Do(req)
	if err != nil {
		return resp, errors.Wrap(err, "failed to execute json request")
	}
	if resp.StatusCode == 500 {
		return nil, errors.Wrap(err, "internal Web3Signer server error")
	} else if resp.StatusCode == 400 {
		return nil, errors.Wrap(err, "bad request format")
	}
	return resp, err
}

// unmarshalResponse is a utility method for unmarshalling responses.
func (*client) unmarshalResponse(responseBody io.ReadCloser, unmarshalledResponseObject interface{}) error {
	defer closeBody(responseBody)
	if err := json.NewDecoder(responseBody).Decode(&unmarshalledResponseObject); err != nil {
		return errors.Wrap(err, "invalid format, unable to read response body as array of strings")
	}
	return nil
}

// decodeHex a utility method for decoding hex strings may be a duplicate in which case will be removed in the future.
func decodeHex(signature string) ([]byte, error) {
	decoded, err := hex.DecodeString(strings.TrimPrefix(signature, "0x"))
	if err != nil {
		return nil, errors.Wrap(err, "invalid format, failed to unmarshal json response")
	}
	return decoded, nil
}

// closeBody a utility method to wrap an error for closing
func closeBody(body io.Closer) {
	if err := body.Close(); err != nil {
		log.Errorf("could not close response body: %v", err)
	}
}
