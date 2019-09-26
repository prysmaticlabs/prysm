package accounts

import (
	"encoding/json"
	"io"
	"io/ioutil"
)

type unencryptedKeysContainer struct {
	Keys []*unencryptedKeys `json:"keys"`
}

type unencryptedKeys struct {
	ValidatorKey  []byte `json:"validator_key"`
	WithdrawalKey []byte `json:"withdrawal_key"`
}

func parseUnencryptedKeysFile(r io.Reader) ([][]byte, [][]byte, error) {
	encoded, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, nil, err
	}
	var ctnr *unencryptedKeysContainer
	if err := json.Unmarshal(encoded, &ctnr); err != nil {
		return nil, nil, err
	}
	validatorKeys := make([][]byte, 0)
	withdrawalKeys := make([][]byte, 0)
	for _, item := range ctnr.Keys {
		validatorKeys = append(validatorKeys, item.ValidatorKey)
		withdrawalKeys = append(withdrawalKeys, item.WithdrawalKey)
	}
	return validatorKeys, withdrawalKeys, nil
}
