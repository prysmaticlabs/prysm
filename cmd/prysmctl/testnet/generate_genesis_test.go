package testnet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/crypto/bls"
	"github.com/prysmaticlabs/prysm/v3/runtime/interop"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func Test_genesisStateFromJSONValidators(t *testing.T) {
	numKeys := 5
	jsonData := createGenesisDepositData(t, numKeys)
	jsonInput, err := json.Marshal(jsonData)
	require.NoError(t, err)
	ctx := context.Background()
	genesisState, err := genesisStateFromJSONValidators(
		ctx, bytes.NewReader(jsonInput), 0, /* genesis time defaults to time.Now() */
	)
	require.NoError(t, err)
	for i, val := range genesisState.Validators {
		assert.DeepEqual(t, fmt.Sprintf("%#x", val.PublicKey), jsonData[i].PubKey)
	}
}

func createGenesisDepositData(t *testing.T, numKeys int) []*depositDataJSON {
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
	jsonData := make([]*depositDataJSON, numKeys)
	for i := 0; i < numKeys; i++ {
		dataRoot, err := dataList[i].HashTreeRoot()
		require.NoError(t, err)
		jsonData[i] = &depositDataJSON{
			PubKey:                fmt.Sprintf("%#x", dataList[i].PublicKey),
			Amount:                dataList[i].Amount,
			WithdrawalCredentials: fmt.Sprintf("%#x", dataList[i].WithdrawalCredentials),
			DepositDataRoot:       fmt.Sprintf("%#x", dataRoot),
			Signature:             fmt.Sprintf("%#x", dataList[i].Signature),
		}
	}
	return jsonData
}
