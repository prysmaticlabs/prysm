package remote_web3signer

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/ioutil"
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

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type client struct {
	BasePath   string
	restClient HTTPClient
}

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

type SignRequest struct {
	Type         string        `json:"type"`
	ForkInfo     *ForkInfo     `json:"fork_info"`
	SigningRoot  string        `json:"signingRoot"`
	RandaoReveal *RandaoReveal `json:"randao_reveal"`
}

type ForkInfo struct {
	Fork                  *Fork  `json:"fork"`
	GenesisValidatorsRoot string `json:"genesis_validators_root"`
}

type Fork struct {
	PreviousVersion string `json:"previous_version"`
	CurrentVersion  string `json:"current_version"`
	Epoch           string `json:"epoch"`
}

type RandaoReveal struct {
	Epoch string `json:"epoch"`
}

type signResponse struct {
	Signature string `json:"signature"`
}

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
	defer closeBody(resp.Body)
	jsonDataFromHttp, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}
	signResp := &signResponse{}
	if err := json.Unmarshal(jsonDataFromHttp, signResp); err != nil {
		return nil, errors.Wrap(err, "invalid format, failed to unmarshal json response")
	}
	decoded, err := decodeHex(signResp.Signature)
	if err != nil {
		return nil, err
	}
	blsSig, err := bls.SignatureFromBytes([]byte(decoded))
	if err != nil {
		return nil, err
	}
	return blsSig, nil
}

func (client *client) GetPublicKeys() ([][]byte, error) {
	const requestPath = "/publicKeys"
	resp, err := client.doRequest(http.MethodGet, client.BasePath+requestPath, nil)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp.Body)
	var publicKeys []string
	if err := json.NewDecoder(resp.Body).Decode(&publicKeys); err != nil {
		return nil, errors.Wrap(err, "invalid format, unable to read response body as array of strings")
	}
	decodedKeys := make([][]byte, len(publicKeys))
	var errorMessage string
	for i, value := range publicKeys {
		decodedKey, err := decodeHex(value)
		if err != nil {
			if errorMessage == "" {
				errorMessage = "failed to decode from Hex from the following public keys: "
			}
			errorMessage += value + " "
			continue
		}
		decodedKeys[i] = decodedKey[:]
	}
	if errorMessage != "" {
		return nil, errors.New(errorMessage)
	}
	return decodedKeys, nil
}

func (client *client) ReloadSignerKeys() error {
	const requestPath = "/reload"
	resp, err := client.doRequest(http.MethodPost, client.BasePath+requestPath, nil)
	if err != nil {
		return err
	}
	defer closeBody(resp.Body)
	return nil
}

func (client *client) GetServerStatus() (string, error) {
	const requestPath = "/upcheck"
	resp, err := client.doRequest(http.MethodGet, client.BasePath+requestPath, nil)
	if err != nil {
		return "", err
	}
	defer closeBody(resp.Body)
	var status string
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return "", errors.Wrap(err, "invalid format, unable to read response body as a string")
	}
	return status, nil
}

func (client *client) doRequest(httpMethod string, fullPath string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(httpMethod, fullPath, body)
	if err != nil {
		return nil, errors.Wrap(err, "invalid format, failed to create new Post Request Object")
	}
	resp, err := client.restClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute json request")
	}
	return resp, nil
}

func decodeHex(signature string) ([]byte, error) {
	decoded, err := hex.DecodeString(strings.TrimPrefix(signature, "0x"))
	if err != nil {
		return nil, errors.Wrap(err, "invalid format, failed to unmarshal json response")
	}
	return decoded, nil
}
func closeBody(body io.Closer) {
	if err := body.Close(); err != nil {
		log.Errorf("could not close response body: %v", err)
	}
}
