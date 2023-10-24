package apimiddleware

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/api/gateway/apimiddleware"
)

// ValidatorEndpointFactory creates endpoints used for running validator API calls through the API Middleware.
type ValidatorEndpointFactory struct {
}

func (f *ValidatorEndpointFactory) IsNil() bool {
	return f == nil
}

// Paths is a collection of all valid validator API paths.
func (*ValidatorEndpointFactory) Paths() []string {
	return []string{
		"/eth/v1/keystores",
		"/eth/v1/validator/{pubkey}/feerecipient",
	}
}

// Create returns a new endpoint for the provided API path.
func (*ValidatorEndpointFactory) Create(path string) (*apimiddleware.Endpoint, error) {
	endpoint := apimiddleware.DefaultEndpoint()
	switch path {
	case "/eth/v1/keystores":
		endpoint.GetResponse = &ListKeystoresResponseJson{}
		endpoint.PostRequest = &ImportKeystoresRequestJson{}
		endpoint.PostResponse = &ImportKeystoresResponseJson{}
		endpoint.DeleteRequest = &DeleteKeystoresRequestJson{}
		endpoint.DeleteResponse = &DeleteKeystoresResponseJson{}
	case "/eth/v1/validator/{pubkey}/feerecipient":
		endpoint.GetResponse = &GetFeeRecipientByPubkeyResponseJson{}
		endpoint.PostRequest = &SetFeeRecipientByPubkeyRequestJson{}
		endpoint.DeleteRequest = &DeleteFeeRecipientByPubkeyRequestJson{}
	default:
		return nil, errors.New("invalid path")
	}
	endpoint.Path = path
	return &endpoint, nil
}
