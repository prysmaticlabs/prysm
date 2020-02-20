package beacon

import (
	"sync"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestInfostream_EpochToTimestamp(t *testing.T) {
	params.UseMainnetConfig()
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
		if timestamp != test.timestamp {
			t.Errorf("Incorrect timestamp: expected %v, received %v", test.timestamp, timestamp)
		}
	}
}

func TestInfostream_HandleSetValidatorKeys(t *testing.T) {
	params.UseMainnetConfig()
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
	}
	for _, test := range tests {
		is.handleSetValidatorKeys(test.reqPubKeys)
		if len(is.pubKeys) != len(test.reqPubKeys) {
			t.Errorf("Incorrect number of keys: expected %v, received %v", len(test.reqPubKeys), len(is.pubKeys))
		}
	}
}

func TestInfostream_HandleAddValidatorKeys(t *testing.T) {
	params.UseMainnetConfig()
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
	}
	for _, test := range tests {
		is.handleSetValidatorKeys(test.initialPubKeys)
		is.handleAddValidatorKeys(test.reqPubKeys)
		if len(is.pubKeys) != test.finalLen {
			t.Errorf("Incorrect number of keys: expected %v, received %v", len(is.pubKeys), test.finalLen)
		}
	}
}

func TestInfostream_HandleRemoveValidatorKeys(t *testing.T) {
	params.UseMainnetConfig()
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
	}
	for _, test := range tests {
		is.handleSetValidatorKeys(test.initialPubKeys)
		is.handleRemoveValidatorKeys(test.reqPubKeys)
		if len(is.pubKeys) != test.finalLen {
			t.Errorf("Incorrect number of keys: expected %v, received %v", len(is.pubKeys), test.finalLen)
		}
	}
}
