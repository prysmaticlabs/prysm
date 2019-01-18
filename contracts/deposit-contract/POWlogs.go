package depositContract

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	contracts "github.com/prysmaticlabs/prysm/contracts/validator-registration-contract"
)

// UnpackDepositLogData unpacks the data from a deposit log using the ABI decoder.
func UnpackDepositLogData(data []byte) (depositRoot [32]byte, datad []byte, merkleTreeIndex []byte, err error) {
	reader := bytes.NewReader([]byte(DepositContractABI))
	contractAbi, err := abi.JSON(reader)
	if err != nil {
		return [32]byte{}, nil, nil, fmt.Errorf("unable to generate contract abi: %v", err)
	}

	unpackedLogs := []interface{}{
		&depositRoot,
		&datad,
		&merkleTreeIndex,
	}
	if err := contractAbi.Unpack(&unpackedLogs, "Deposit", data); err != nil {
		return [32]byte{}, nil, nil, fmt.Errorf("unable to unpack logs: %v", err)
	}

	return depositRoot, datad, merkleTreeIndex, nil
}

// UnpackChainStartLogData unpacks the data from a chain start log using the ABI decoder.
func UnpackChainStartLogData(data []byte) ([]byte, error) {
	reader := bytes.NewReader([]byte(contracts.ValidatorRegistrationABI))
	contractAbi, err := abi.JSON(reader)
	if err != nil {
		return nil, fmt.Errorf("unable to generate contract abi: %v", err)
	}
	unpackedLogs := []*[]byte{
		&[]byte{},
	}
	if err := contractAbi.Unpack(&unpackedLogs, "ChainStart", data); err != nil {
		return nil, fmt.Errorf("unable to unpack logs: %v", err)
	}
	timestamp := *unpackedLogs[0]
	return timestamp, nil
}
