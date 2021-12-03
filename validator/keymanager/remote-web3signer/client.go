package remote_web3signer

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"net/http"
	"net/url"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type client struct {
	BasePath   string
	APIPath    string
	restClient HTTPClient
}

func newClient(endpoint string) (*client, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	return &client{
		BasePath:   u.Host,
		APIPath:    u.Path,
		restClient: &http.Client{},
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

//func (client *client) Sign(pubKey string, request *SignRequest) (bls.Signature, error) {
//	requestPath := "sign/" + pubKey
//
//	jsonRequest, err := json.Marshal(request)
//
//	if err != nil {
//		// return error
//	}
//
//	resp, err := http.Post(
//		client.constructFullURL(requestPath),
//		"application/json",
//		bytes.NewBuffer(jsonRequest),
//	)
//
//	if err != nil {
//		// return error
//	}
//
//	defer func() {
//		if err := resp.Body.Close(); err != nil {
//
//		}
//	}()
//
//	// is this how we are supposed to do it?
//	jsonDataFromHttp, err := ioutil.ReadAll(resp.Body)
//	if err != nil {
//		//panic(err)
//	}
//
//	signResp := &signResponse{}
//	if err := json.Unmarshal(jsonDataFromHttp, signResp); err != nil {
//		return nil, errors.Wrap(err, "???")
//	}
//
//	// should hex.decode, trim prefix
//	blsSig, err := bls.SignatureFromBytes([]byte(signResp.Signature))
//	if err != nil {
//		//panic(err)
//	}
//
//	return blsSig, nil
//}

func (client *client) GetPublicKeys() ([]string, error) {
	const requestPath = "publicKeys"
	req, err := http.NewRequest(http.MethodGet, client.constructFullURL(requestPath), nil)
	if err != nil {
		fmt.Printf(" error1:  %d", err)

	}

	resp, err := client.restClient.Do(req)
	if err != nil {
		fmt.Printf("error2: %d", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {

		}
	}()
	var publicKeys []string
	if err := json.NewDecoder(resp.Body).Decode(&publicKeys); err != nil {
		return nil, errors.Wrap(err, "??")
	}

	return publicKeys, nil
}

func (client *client) ReloadSignerKeys() error {
	const requestPath = "reload"
	req, err := http.NewRequest(http.MethodPost, client.constructFullURL(requestPath), nil)
	if err != nil {
		fmt.Printf(" error1:  %d", err)
		return err
	}
	resp, err := client.restClient.Do(req)
	if err != nil {
		fmt.Printf("error2: %d", err)
		return err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {

		}
	}()

	return nil
}

func (client *client) GetServerStatus() (string, error) {
	const requestPath = "upcheck"
	req, err := http.NewRequest(http.MethodGet, client.constructFullURL(requestPath), nil)
	if err != nil {
		fmt.Printf(" error1:  %d", err)
		return "", err
	}
	resp, err := client.restClient.Do(req)
	if err != nil {
		fmt.Printf("error2: %d", err)
		return "", err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {

		}
	}()
	var status string
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return "", errors.Wrap(err, "??")
	}
	return status, nil
}

func (client *client) constructFullURL(actionPath string) string {
	return client.BasePath + client.APIPath + actionPath
}
