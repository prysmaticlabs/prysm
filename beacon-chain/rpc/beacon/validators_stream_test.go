package beacon

import (
	"sync"
	"testing"

	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func TestInfostream_EpochToTimestamp(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	tests := []struct {
		name      string
		epoch     uint64
		timestamp uint64
	}{
		{
			name:      "Genesis",
			epoch:     0,
			timestamp: 0,
		},
		{
			name:      "One",
			epoch:     1,
			timestamp: 384,
		},
		{
			name:      "Two",
			epoch:     2,
			timestamp: 768,
		},
		{
			name:      "OneHundred",
			epoch:     100,
			timestamp: 38400,
		},
	}

	is := &infostream{}
	for _, test := range tests {
		timestamp := is.epochToTimestamp(test.epoch)
		assert.Equal(t, test.timestamp, timestamp, "Incorrect timestamp")
	}
}

func TestInfostream_HandleSetValidatorKeys(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	tests := []struct {
		name       string
		reqPubKeys [][]byte
	}{
		{
			name: "None",
		},
		{
			name:       "One",
			reqPubKeys: [][]byte{{0x01}},
		},
		{
			name:       "Two",
			reqPubKeys: [][]byte{{0x01}, {0x02}},
		},
	}

	is := &infostream{
		pubKeysMutex: &sync.RWMutex{},
		pubKeys:      make([][]byte, 0),
		headFetcher: &mock.ChainService{
			State: testutil.NewBeaconState(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.NoError(t, is.handleSetValidatorKeys(test.reqPubKeys))
			assert.Equal(t, len(test.reqPubKeys), len(is.pubKeys), "Incorrect number of keys")
		})
	}
}

func TestInfostream_HandleAddValidatorKeys(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	tests := []struct {
		name           string
		initialPubKeys [][]byte
		reqPubKeys     [][]byte
		finalLen       int
	}{
		{
			name:     "None",
			finalLen: 0,
		},
		{
			name:       "NoneAddOne",
			reqPubKeys: [][]byte{{0x01}},
			finalLen:   1,
		},
		{
			name:           "OneAddOne",
			initialPubKeys: [][]byte{{0x01}},
			reqPubKeys:     [][]byte{{0x02}},
			finalLen:       2,
		},
		{
			name:           "Duplicate",
			initialPubKeys: [][]byte{{0x01}},
			reqPubKeys:     [][]byte{{0x01}},
			finalLen:       1,
		},
	}

	is := &infostream{
		pubKeysMutex: &sync.RWMutex{},
		pubKeys:      make([][]byte, 0),
		headFetcher: &mock.ChainService{
			State: testutil.NewBeaconState(),
		},
	}
	for _, test := range tests {
		assert.NoError(t, is.handleSetValidatorKeys(test.initialPubKeys))
		assert.NoError(t, is.handleAddValidatorKeys(test.reqPubKeys))
		assert.Equal(t, test.finalLen, len(is.pubKeys), "Incorrect number of keys")
	}
}

func TestInfostream_HandleRemoveValidatorKeys(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	tests := []struct {
		name           string
		initialPubKeys [][]byte
		reqPubKeys     [][]byte
		finalLen       int
	}{
		{
			name:     "None",
			finalLen: 0,
		},
		{
			name:           "OneRemoveNone",
			initialPubKeys: [][]byte{{0x01}},
			finalLen:       1,
		},
		{
			name:           "NoneRemoveOne",
			initialPubKeys: [][]byte{},
			reqPubKeys:     [][]byte{{0x01}},
			finalLen:       0,
		},
		{
			name:           "TwoRemoveOne",
			initialPubKeys: [][]byte{{0x01, 0x02}},
			reqPubKeys:     [][]byte{{0x01}},
			finalLen:       1,
		},
	}

	is := &infostream{
		pubKeysMutex: &sync.RWMutex{},
		pubKeys:      make([][]byte, 0),
		headFetcher: &mock.ChainService{
			State: testutil.NewBeaconState(),
		},
	}
	for _, test := range tests {
		assert.NoError(t, is.handleSetValidatorKeys(test.initialPubKeys))
		is.handleRemoveValidatorKeys(test.reqPubKeys)
		assert.Equal(t, test.finalLen, len(is.pubKeys), "Incorrect number of keys")
	}
}
