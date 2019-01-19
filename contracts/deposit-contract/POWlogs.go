package depositcontract

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// UnpackDepositLogData unpacks the data from a deposit log using the ABI decoder.
func UnpackDepositLogData(data []byte) (depositRoot [32]byte, depositData []byte, merkleTreeIndex []byte, err error) {
	reader := bytes.NewReader([]byte(DepositContractABI))
	contractAbi, err := abi.JSON(reader)
	if err != nil {
		return [32]byte{}, nil, nil, fmt.Errorf("unable to generate contract abi: %v", err)
	}

	unpackedLogs := []interface{}{
		&depositRoot,
		&depositData,
		&merkleTreeIndex,
	}
	if err := contractAbi.Unpack(&unpackedLogs, "Deposit", data); err != nil {
		return [32]byte{}, nil, nil, fmt.Errorf("unable to unpack logs: %v", err)
	}

	return depositRoot, depositData, merkleTreeIndex, nil
}

// UnpackChainStartLogData unpacks the data from a chain start log using the ABI decoder.
func UnpackChainStartLogData(data []byte) (depositRoot [32]byte, timestamp []byte, err error) {
	reader := bytes.NewReader([]byte(DepositContractABI))
	contractAbi, err := abi.JSON(reader)
	if err != nil {
		return [32]byte{}, nil, fmt.Errorf("unable to generate contract abi: %v", err)
	}
	unpackedLogs := []interface{}{
		&depositRoot,
		&timestamp,
	}
	if err := contractAbi.Unpack(&unpackedLogs, "ChainStart", data); err != nil {
		return [32]byte{}, nil, fmt.Errorf("unable to unpack logs: %v", err)
	}

	return depositRoot, timestamp, nil
}
