package validator

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/api/client"
	"github.com/prysmaticlabs/prysm/v4/validator/rpc/apimiddleware"
)

const (
	localKeysPath    = "/eth/v1/keystores"
	remoteKeysPath   = "/eth/v1/remotekeys"
	feeRecipientPath = "/eth/v1/validator/{pubkey}/feerecipient"
)

// ValidatorAPIClient is a wrapper with beacon API functions on the client
type ValidatorAPIClient struct {
	*client.Client
}

// NewValidatorAPIClient returns a new alidatorAPIClient that includes functions for rest calls to keymanager APIs.
func NewValidatorAPIClient(host string, opts ...client.ClientOpt) (*ValidatorAPIClient, error) {
	c, err := client.NewClient(host, opts...)
	if err != nil {
		return nil, err
	}
	return &ValidatorAPIClient{c}, nil
}

// GetListOfValidators gets the currently known validators in hex format on the validator client whether on the web3signer or the local keystores.
func (c *ValidatorAPIClient) GetListOfValidators(ctx context.Context) ([]string, error) {
	localBytes, err := c.Get(ctx, localKeysPath, client.WithTokenAuthorization(c.Token()))
	if err != nil {
		return nil, err
	}
	jsonlocal := &apimiddleware.ListKeystoresResponseJson{}
	if err := json.Unmarshal(localBytes, jsonlocal); err != nil {
		return nil, errors.Wrap(err, "failed to parse local list keystores")
	}
	remoteBytes, err := c.Get(ctx, remoteKeysPath, client.WithTokenAuthorization(c.Token()))
	if err != nil {
		if !strings.Contains(err.Error(), "Prysm Wallet is not of type Web3Signer") {
			return nil, err
		}
	}
	jsonremote := &apimiddleware.ListRemoteKeysResponseJson{}
	if len(remoteBytes) != 0 {
		if err := json.Unmarshal(remoteBytes, jsonremote); err != nil {
			return nil, errors.Wrap(err, "failed to parse remote list keystores")
		}
	}
	if len(jsonlocal.Keystores) == 0 && len(jsonremote.Keystores) == 0 {
		return nil, errors.New("there are no local keys or remote keys on the validator")
	}

	hexKeys := make(map[string]bool)

	for index := range jsonlocal.Keystores {
		hexKeys[jsonlocal.Keystores[index].ValidatingPubkey] = true
	}
	for index := range jsonremote.Keystores {
		hexKeys[jsonremote.Keystores[index].Pubkey] = true
	}
	keys := make([]string, 0)
	for k := range hexKeys {
		keys = append(keys, k)
	}
	return keys, nil
}

// GetListOfFeeRecipients takes a list of validators in hex format and returns an equal length list of fee recipients in hex format.
func (c *ValidatorAPIClient) GetListOfFeeRecipients(ctx context.Context, validators []string) ([]string, error) {
	feeRecipients := make([]string, len(validators))
	for index, validator := range validators {
		path := strings.Replace(feeRecipientPath, "{pubkey}", validator, 1)
		b, err := c.Get(ctx, path, client.WithTokenAuthorization(c.Token()))
		if err != nil {
			return nil, err
		}
		feejson := &apimiddleware.GetFeeRecipientByPubkeyResponseJson{}
		if err := json.Unmarshal(b, feejson); err != nil {
			return nil, errors.Wrap(err, "failed to parse fee recipient")
		}
		if feejson.Data == nil {
			continue
		}
		feeRecipients[index] = feejson.Data.Ethaddress
	}
	return feeRecipients, nil
}
