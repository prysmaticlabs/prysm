package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/interop"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func Test_genesisStateFromJSONValidators(t *testing.T) {
	jsonData, depositDataList := createGenesisDepositData(t)
	jsonInput, err := json.Marshal(jsonData)
	require.NoError(t, err)
	genesisState, err := genesisStateFromJSONValidators(
		bytes.NewReader(jsonInput), 0, /* genesis time defaults to time.Now() */
	)
	require.NoError(t, err)
	for i, val := range genesisState.Validators {
		assert.DeepEqual(t, val.PublicKey, depositDataList[i].PublicKey)
	}
}

func createGenesisDepositData(t *testing.T) ([]*GenesisValidator, []*ethpb.Deposit_Data) {
	numKeys := 5
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
	jsonData := make([]*GenesisValidator, numKeys)
	for i := 0; i < numKeys; i++ {
		data := dataList[i]
		enc, err := data.MarshalSSZ()
		require.NoError(t, err)
		jsonData[i] = &GenesisValidator{
			DepositData: fmt.Sprintf("%#x", enc),
		}
	}
	return jsonData, dataList
}
