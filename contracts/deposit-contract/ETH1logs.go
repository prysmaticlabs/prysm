package depositcontract

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/pkg/errors"
)

// UnpackDepositLogData unpacks the data from a deposit log using the ABI decoder.
func UnpackDepositLogData(data []byte) (pubkey []byte, withdrawalCredentials []byte, amount []byte,
	signature []byte, index []byte, err error) {
	reader := bytes.NewReader([]byte(DepositContractABI))
	contractAbi, err := abi.JSON(reader)
	if err != nil {
		return nil, nil, nil, nil, nil, errors.Wrap(err, "unable to generate contract abi")
	}

	unpackedLogs := []interface{}{
		&pubkey,
		&withdrawalCredentials,
		&amount,
		&signature,
		&index,
	}
	if err := contractAbi.Unpack(&unpackedLogs, "DepositEvent", data); err != nil {
		return nil, nil, nil, nil, nil, errors.Wrap(err, "unable to unpack logs")
	}

	return pubkey, withdrawalCredentials, amount, signature, index, nil
}
