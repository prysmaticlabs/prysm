package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/interop"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func Test_genesisStateFromJSONValidators(t *testing.T) {
	numKeys := 5
	jsonData := createGenesisDepositData(t, numKeys)
	jsonInput, err := json.Marshal(jsonData)
	require.NoError(t, err)
	genesisState, err := genesisStateFromJSONValidators(
		bytes.NewReader(jsonInput), 0, /* genesis time defaults to time.Now() */
	)
	require.NoError(t, err)
	for i, val := range genesisState.Validators {
		assert.DeepEqual(t, fmt.Sprintf("%#x", val.PublicKey), jsonData[i].PubKey)
	}
}

func Test_genesisStateFromJSONFile(t *testing.T) {
	expanded, err := fileutil.ExpandPath("./test-data/deposit_data_32_keys.json")
	require.NoError(t, err)
	inputJSON, err := os.Open(expanded)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, inputJSON.Close())
	}()
	genesisState, err := genesisStateFromJSONValidators(inputJSON, *genesisTime)
	require.NoError(t, err)
	depositRootHex := hexutil.Encode(genesisState.Eth1Data.DepositRoot)
	assert.DeepEqual(t, "0x35d4191c0f6510d5f352868ff9a55c4a82a98a470f365d2495ff632b63c6c94a", depositRootHex)
}

func createGenesisDepositData(t *testing.T, numKeys int) []*DepositDataJSON {
	pubKeys := make([]bls.PublicKey, numKeys)
	privKeys := make([]bls.SecretKey, numKeys)
	for i := 0; i < numKeys; i++ {
		randKey, err := bls.RandKey()
		require.NoError(t, err)
		privKeys[i] = randKey
		pubKeys[i] = randKey.PublicKey()
	}
	dataList, _, err := interop.DepositDataFromKeys(privKeys, pubKeys)
	require.NoError(t, err)
	jsonData := make([]*DepositDataJSON, numKeys)
	for i := 0; i < numKeys; i++ {
		dataRoot, err := dataList[i].HashTreeRoot()
		require.NoError(t, err)
		jsonData[i] = &DepositDataJSON{
			PubKey:                fmt.Sprintf("%#x", dataList[i].PublicKey),
			Amount:                dataList[i].Amount,
			WithdrawalCredentials: fmt.Sprintf("%#x", dataList[i].WithdrawalCredentials),
			DepositDataRoot:       fmt.Sprintf("%#x", dataRoot),
			Signature:             fmt.Sprintf("%#x", dataList[i].Signature),
		}
	}
	return jsonData
}
