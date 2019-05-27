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
	if err := contractAbi.Unpack(&unpackedLogs, "Deposit", data); err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("unable to unpack logs: %v", err)
	}

	return pubkey, withdrawalCredentials, amount, signature, index, nil
}

// UnpackChainStartLogData unpacks the data from a chain start log using the ABI decoder.
func UnpackChainStartLogData(data []byte) (depositRoot [32]byte, depositCount []byte, time []byte, err error) {
	reader := bytes.NewReader([]byte(DepositContractABI))
	contractAbi, err := abi.JSON(reader)
	if err != nil {
		return [32]byte{}, nil, nil, fmt.Errorf("unable to generate contract abi: %v", err)
	}
	unpackedLogs := []interface{}{
		&depositRoot,
		&depositCount,
		&time,
	}
	if err := contractAbi.Unpack(&unpackedLogs, "Eth2Genesis", data); err != nil {
		return [32]byte{}, nil, nil, fmt.Errorf("unable to unpack logs: %v", err)
	}

	return depositRoot, depositCount, time, nil
}
