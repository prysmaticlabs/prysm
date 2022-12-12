package depositsnapshot

import (
	"encoding/hex"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/pkg/errors"
	eth "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"gopkg.in/yaml.v3"
)

type testCase struct {
	DepositData     depositData `yaml:"deposit_data"`
	DepositDataRoot [32]byte    `yaml:"deposit_data_root"`
	Eth1Data        *eth1Data   `yaml:"eth1_data"`
	BlockHeight     uint64      `yaml:"block_height"`
	Snapshot        snapshot    `yaml:"snapshot"`
}

func (tc *testCase) UnmarshalYAML(value *yaml.Node) error {
	raw := struct {
		DepositData     depositData `yaml:"deposit_data"`
		DepositDataRoot string      `yaml:"deposit_data_root"`
		Eth1Data        *eth1Data   `yaml:"eth1_data"`
		BlockHeight     string      `yaml:"block_height"`
		Snapshot        snapshot    `yaml:"snapshot"`
	}{}
	err := value.Decode(&raw)
	if err != nil {
		return err
	}
	tc.DepositDataRoot, err = hexStringToByteArray(raw.DepositDataRoot)
	if err != nil {
		return err
	}
	tc.DepositData = raw.DepositData
	tc.Eth1Data = raw.Eth1Data
	tc.BlockHeight, err = stringToUint64(raw.BlockHeight)
	if err != nil {
		return err
	}
	tc.Snapshot = raw.Snapshot
	return nil
}

type depositData struct {
	Pubkey                []byte `yaml:"pubkey"`
	WithdrawalCredentials []byte `yaml:"withdrawal_credentials"`
	Amount                uint64 `yaml:"amount"`
	Signature             []byte `yaml:"signature"`
}

func (dd *depositData) UnmarshalYAML(value *yaml.Node) error {
	raw := struct {
		Pubkey                string `yaml:"pubkey"`
		WithdrawalCredentials string `yaml:"withdrawal_credentials"`
		Amount                string `yaml:"amount"`
		Signature             string `yaml:"signature"`
	}{}
	err := value.Decode(&raw)
	if err != nil {
		return err
	}
	dd.Pubkey, err = hexStringToBytes(raw.Pubkey)
	if err != nil {
		return err
	}
	dd.WithdrawalCredentials, err = hexStringToBytes(raw.WithdrawalCredentials)
	if err != nil {
		return err
	}
	dd.Amount, err = strconv.ParseUint(raw.Amount, 10, 64)
	if err != nil {
		return err
	}
	dd.Signature, err = hexStringToBytes(raw.Signature)
	if err != nil {
		return err
	}
	return nil
}

type eth1Data struct {
	DepositRoot  [32]byte `yaml:"deposit_root"`
	DepositCount uint64   `yaml:"deposit_count"`
	BlockHash    [32]byte `yaml:"block_hash"`
}

func (ed *eth1Data) UnmarshalYAML(value *yaml.Node) error {
	raw := struct {
		DepositRoot  string `yaml:"deposit_root"`
		DepositCount string `yaml:"deposit_count"`
		BlockHash    string `yaml:"block_hash"`
	}{}
	err := value.Decode(&raw)
	if err != nil {
		return err
	}
	ed.DepositRoot, err = hexStringToByteArray(raw.DepositRoot)
	if err != nil {
		return err
	}
	ed.DepositCount, err = stringToUint64(raw.DepositCount)
	if err != nil {
		return err
	}
	ed.BlockHash, err = hexStringToByteArray(raw.BlockHash)
	if err != nil {
		return err
	}
	return nil
}

type snapshot struct {
	Finalized            [][32]byte `yaml:"finalized"`
	DepositRoot          [32]byte   `yaml:"deposit_root"`
	DepositCount         uint64     `yaml:"deposit_count"`
	ExecutionBlockHash   [32]byte   `yaml:"execution_block_hash"`
	ExecutionBlockHeight uint64     `yaml:"execution_block_height"`
}

func (sd *snapshot) UnmarshalYAML(value *yaml.Node) error {
	raw := struct {
		Finalized            []string `yaml:"finalized"`
		DepositRoot          string   `yaml:"deposit_root"`
		DepositCount         string   `yaml:"deposit_count"`
		ExecutionBlockHash   string   `yaml:"execution_block_hash"`
		ExecutionBlockHeight string   `yaml:"execution_block_height"`
	}{}
	err := value.Decode(&raw)
	if err != nil {
		return err
	}
	sd.Finalized = make([][32]byte, len(raw.Finalized))
	for i, finalized := range raw.Finalized {
		sd.Finalized[i], err = hexStringToByteArray(finalized)
		if err != nil {
			return err
		}
	}
	sd.DepositRoot, err = hexStringToByteArray(raw.DepositRoot)
	if err != nil {
		return err
	}
	sd.DepositCount, err = stringToUint64(raw.DepositCount)
	if err != nil {
		return err
	}
	sd.ExecutionBlockHash, err = hexStringToByteArray(raw.ExecutionBlockHash)
	if err != nil {
		return err
	}
	sd.ExecutionBlockHeight, err = stringToUint64(raw.ExecutionBlockHeight)
	if err != nil {
		return err
	}
	return nil
}

func readTestCases(filename string) ([]testCase, error) {
	var testCases []testCase
	file, err := os.ReadFile(filename)
	if err != nil {
		return []testCase{}, err
	}
	err = yaml.Unmarshal(file, &testCases)
	if err != nil {
		return []testCase{}, err
	}
	return testCases, nil
}

func TestRead(t *testing.T) {
	tcs, err := readTestCases("test_cases.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range tcs {
		t.Log(tc)
	}
}

func hexStringToByteArray(s string) (b [32]byte, err error) {
	var raw []byte
	raw, err = hexStringToBytes(s)
	if err != nil {
		return
	}
	if len(raw) != 32 {
		err = errors.New("invalid hex string length")
		return
	}
	copy(b[:], raw[:32])
	return
}

func hexStringToBytes(s string) (b []byte, err error) {
	b, err = hex.DecodeString(strings.TrimPrefix(s, "0x"))
	return
}

func stringToUint64(s string) (uint64, error) {
	value, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return value, nil
}

func TestDepositCases(t *testing.T) {
	tree := New()
	testCases, err := readTestCases("test_cases.yaml")
	assert.NoError(t, err)
	for _, c := range testCases {
		err = tree.pushLeaf(c.DepositDataRoot)
		assert.NoError(t, err)
		assert.Equal(t, c.Snapshot.DepositRoot, c.Eth1Data.DepositRoot)
		assert.Equal(t, tree.getRoot(), c.Eth1Data.DepositRoot)
	}
}

func TestFinalization(t *testing.T) {
	tree := New()
	testCases, err := readTestCases("test_cases.yaml")
	assert.NoError(t, err)
	for _, c := range testCases[:128] {
		err = tree.pushLeaf(c.DepositDataRoot)
		assert.NoError(t, err)
	}
	originalRoot := tree.getRoot()
	assert.DeepEqual(t, originalRoot, testCases[127].Eth1Data.DepositRoot)
	tree.finalize(&eth.Eth1Data{
		DepositRoot:  testCases[100].Eth1Data.DepositRoot[:],
		DepositCount: testCases[100].Eth1Data.DepositCount,
		BlockHash:    testCases[100].Eth1Data.BlockHash[:],
	}, testCases[100].BlockHeight)
	assert.Equal(t, tree.getRoot(), originalRoot)
	s, err := tree.getSnapshot()
	assert.NoError(t, err)
	assert.Equal(t, s, testCases[100].Snapshot)
}

func TestSnapshotCases(t *testing.T) {
	tree := New()
	testCases, err := readTestCases("test_cases.yaml")
	assert.NoError(t, err)
	for _, c := range testCases {
		err = tree.pushLeaf(c.DepositDataRoot)
		assert.NoError(t, err)
	}
	for _, c := range testCases {
		tree.finalize(&eth.Eth1Data{
			DepositRoot:  c.Eth1Data.DepositRoot[:],
			DepositCount: c.Eth1Data.DepositCount,
			BlockHash:    c.Eth1Data.BlockHash[:],
		}, c.BlockHeight)
		s, err := tree.getSnapshot()
		assert.NoError(t, err)
		assert.Equal(t, s, c.Snapshot)
	}
}

func TestEmptyTreeSnapshot(t *testing.T) {
	_, err := New().getSnapshot()
	assert.ErrorContains(t, "empty execution block", err)
}

func TestInvalidSnapshot(t *testing.T) {
	invalidSnapshot := DepositTreeSnapshot{
		finalized:            nil,
		depositRoot:          Zerohashes[0],
		depositCount:         0,
		executionBlockHash:   Zerohashes[0],
		executionBlockHeight: 0,
	}
	_, err := fromSnapshot(invalidSnapshot)
	assert.ErrorContains(t, "snapshot root is invalid", err)
}
