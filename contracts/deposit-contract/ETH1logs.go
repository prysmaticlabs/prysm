package depositcontract

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// UnpackDepositLogData unpacks the data from a deposit log using the ABI decoder.
func UnpackDepositLogData(data []byte) (pubkey [48]byte, withdrawalCredentials [32]byte, amount [8]byte,
	signature [96]byte, index [8]byte, err error) {
	reader := bytes.NewReader([]byte(DepositContractABI))
	contractAbi, err := abi.JSON(reader)
	if err != nil {
		return [48]byte{}, [32]byte{}, [8]byte{}, [96]byte{}, [8]byte{}, fmt.Errorf("unable to generate contract abi: %v", err)
	}

	unpackedLogs := []interface{}{
		&pubkey,
		&withdrawalCredentials,
		&amount,
		&signature,
		&index,
	}
	if err := contractAbi.Unpack(&unpackedLogs, "Deposit", data); err != nil {
		return [48]byte{}, [32]byte{}, [8]byte{}, [96]byte{}, [8]byte{}, fmt.Errorf("unable to unpack logs: %v", err)
	}

	return pubkey, withdrawalCredentials, amount, signature, index, nil
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
