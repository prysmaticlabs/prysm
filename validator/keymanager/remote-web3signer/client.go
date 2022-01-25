package remote_web3signer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	v1 "github.com/prysmaticlabs/prysm/validator/keymanager/remote-web3signer/v1"
)

const (
	ethApiNamespace = "/api/v1/eth2/sign/"
	maxTimeout      = 3 * time.Second
)

type SignRequestJson []byte

// httpSignerClient defines the interface for interacting with a remote web3signer.
type httpSignerClient interface {
	Sign(ctx context.Context, pubKey string, request SignRequestJson) (bls.Signature, error)
	GetPublicKeys(ctx context.Context, url string) ([][48]byte, error)
}

// apiClient a wrapper object around web3signer APIs. API docs found here https://consensys.github.io/web3signer/web3signer-eth2.html.
type apiClient struct {
	BaseURL    *url.URL
	restClient *http.Client
}

// newApiClient method instantiates a new apiClient object.
func newApiClient(baseEndpoint string) (*apiClient, error) {
	u, err := url.Parse(baseEndpoint)
	if err != nil {
		return nil, errors.Wrap(err, "invalid format, unable to parse url")
	}
	return &apiClient{
		BaseURL: u,
		restClient: &http.Client{
			Timeout: maxTimeout,
		},
	}, nil
}

// Sign is a wrapper method around the web3signer sign api.
func (client *apiClient) Sign(_ context.Context, pubKey string, request SignRequestJson) (bls.Signature, error) {
	requestPath := ethApiNamespace + pubKey
	resp, err := client.doRequest(http.MethodPost, client.BaseURL.String()+requestPath, bytes.NewBuffer(request))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("public key not found")
	}
	if resp.StatusCode == http.StatusPreconditionFailed {
		return nil, fmt.Errorf("signing operation failed due to slashing protection rules,  Signing Request URL: %v, Signing Request Body: %v, Full Response: %v", client.BaseURL.String()+requestPath, string(request), resp)
	}
	signResp := &v1.SignResponse{}
	if err := client.unmarshalResponse(resp.Body, &signResp); err != nil {
		return nil, err
	}
	decoded, err := hexutil.Decode(signResp.Signature)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode signature")
	}
	return bls.SignatureFromBytes(decoded)
}

// GetPublicKeys is a wrapper method around the web3signer publickeys api (this may be removed in the future or moved to another location due to its usage).
func (client *apiClient) GetPublicKeys(_ context.Context, url string) ([][fieldparams.BLSPubkeyLength]byte, error) {
	resp, err := client.doRequest(http.MethodGet, url, nil /* no body needed on get request */)
	if err != nil {
		return nil, err
	}
	var publicKeys []string
	if err := client.unmarshalResponse(resp.Body, &publicKeys); err != nil {
		return nil, err
	}
	decodedKeys := make([][fieldparams.BLSPubkeyLength]byte, len(publicKeys))
	var errorKeyPositions string
	for i, value := range publicKeys {
		decodedKey, err := hexutil.Decode(value)
		if err != nil {
			errorKeyPositions += fmt.Sprintf("%v, ", i)
			continue
		}
		decodedKeys[i] = bytesutil.ToBytes48(decodedKey)
	}
	if errorKeyPositions != "" {
		return nil, errors.New("failed to decode from Hex from the following public key index locations: " + errorKeyPositions)
	}
	return decodedKeys, nil
}

// ReloadSignerKeys is a wrapper method around the web3signer reload api.
func (client *apiClient) ReloadSignerKeys(_ context.Context) error {
	const requestPath = "/reload"
	if _, err := client.doRequest(http.MethodPost, client.BaseURL.String()+requestPath, nil); err != nil {
		return err
	}
	return nil
}

// GetServerStatus is a wrapper method around the web3signer upcheck api
func (client *apiClient) GetServerStatus(_ context.Context) (string, error) {
	const requestPath = "/upcheck"
	resp, err := client.doRequest(http.MethodGet, client.BaseURL.String()+requestPath, nil /* no body needed on get request */)
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
func (client *apiClient) doRequest(httpMethod, fullPath string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(httpMethod, fullPath, body)
	if err != nil {
		return nil, errors.Wrap(err, "invalid format, failed to create new Post Request Object")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.restClient.Do(req)
	if err != nil {
		return resp, errors.Wrap(err, "failed to execute json request")
	}
	if resp.StatusCode == http.StatusInternalServerError {
		b, err := io.ReadAll(body)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read body")
		}
		return nil, fmt.Errorf("internal Web3Signer server error, Signing Request URL: %v, Signing Request Body: %s, Full Response: %v", fullPath, string(b), resp)
	} else if resp.StatusCode == http.StatusBadRequest {
		b, err := io.ReadAll(body)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read body")
		}
		return nil, fmt.Errorf("bad request format, Signing Request URL: %v, Signing Request Body: %v, Full Response: %v", fullPath, string(b), resp)
	}
	return resp, err
}

// unmarshalResponse is a utility method for unmarshalling responses.
func (*apiClient) unmarshalResponse(responseBody io.ReadCloser, unmarshalledResponseObject interface{}) error {
	defer closeBody(responseBody)
	gotRes, err := ioutil.ReadAll(responseBody)
	if err != nil {
		return err
	}
	fmt.Printf("HTTP Response: %s", string(gotRes))
	if err := json.NewDecoder(bytes.NewBuffer(gotRes)).Decode(&unmarshalledResponseObject); err != nil {
		return errors.Wrap(err, "invalid format, unable to read response body as array of strings")
	}
	return nil
}

// closeBody a utility method to wrap an error for closing
func closeBody(body io.Closer) {
	if err := body.Close(); err != nil {
		log.Errorf("could not close response body: %v", err)
	}
}
