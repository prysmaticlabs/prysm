package depositcontract

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// UnpackDepositLogData unpacks the data from a deposit log using the ABI decoder.
func UnpackDepositLogData(data []byte) (pubkey []byte, withdrawalCredentials []byte, amount []byte,
	signature []byte, index []byte, err error) {
	reader := bytes.NewReader([]byte(DepositContractABI))
	contractAbi, err := abi.JSON(reader)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("unable to generate contract abi: %v", err)
	}

	unpackedLogs := []interface{}{
		&pubkey,
		&withdrawalCredentials,
		&amount,
		&signature,
		&index,
	}
	if err := contractAbi.Unpack(&unpackedLogs, "DepositEvent", data); err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("unable to unpack logs: %v", err)
	}

	return pubkey, withdrawalCredentials, amount, signature, index, nil
}
