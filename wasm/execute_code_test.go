package wasm

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v2"
)

const testsPath = "tests"

type testCase struct {
	Description string
	Script      []byte
	PreState    [32]byte
	Block       []byte
	PostState   [32]byte
	Deposits    []Deposit
	Error       int
}

type YamlDeposit struct {
	PubKey                string `yaml:"pubKey"`
	WithdrawalCredentials string `yaml:"withdrawalCredentials"`
	Amount                uint64 `yaml:"amount"`
}

type YamlFile []struct {
	Description string        `yaml:"description"`
	Script      string        `yaml:"script"`
	PreState    string        `yaml:"pre_state"`
	Block       string        `yaml:"block,omitempty"`
	PostState   string        `yaml:"post_state"`
	Deposits    []YamlDeposit `yaml:"deposits,omitempty"`
	Error       int           `yaml:"error,omitempty"`
}

func readHex(t *testing.T, hexString string) []byte {
	result, err := hex.DecodeString(hexString)
	if err != nil {
		t.Fatalf("can't read hex string: %v err: %v", hexString, err)
	}
	return result
}

func readHex32(t *testing.T, hexString string) [32]byte {
	slice := readHex(t, hexString)
	var result [32]byte
	n := copy(result[:], slice)
	if n != 32 {
		t.Fatalf("can't read 32 bytes. n = %v", n)
	}
	return result
}

func readHex48(t *testing.T, hexString string) [48]byte {
	slice := readHex(t, hexString)
	var result [48]byte
	n := copy(result[:], slice)
	if n != 48 {
		t.Fatalf("can't read 48 bytes. n = %v", n)
	}
	return result
}

func readScript(t *testing.T, fileNameOrHex string) []byte {
	fileName := path.Join(testsPath, fileNameOrHex)
	if _, err := os.Stat(fileName); !os.IsNotExist(err) {
		wasm, err := ioutil.ReadFile(fileName)
		if err != nil {
			t.Fatalf("can't read file. %v", err)
		}
		return wasm
	}
	return readHex(t, fileNameOrHex)
}

func readDeposits(t *testing.T, yamlDeposits []YamlDeposit) []Deposit {
	var result []Deposit
	for _, yamlDeposit := range yamlDeposits {
		result = append(result, Deposit{
			PubKey:                readHex48(t, yamlDeposit.PubKey),
			WithdrawalCredentials: readHex48(t, yamlDeposit.WithdrawalCredentials),
			Amount:                yamlDeposit.Amount,
		})
	}
	return result
}

func readYaml(t *testing.T, yamlFileName string) []testCase {
	yamlBytes, err := ioutil.ReadFile(yamlFileName)
	if err != nil {
		t.Fatalf("can't read the %v: %v", yamlFileName, err)
	}
	var yamlFile YamlFile
	if err := yaml.Unmarshal(yamlBytes, &yamlFile); err != nil {
		t.Fatalf("can't unmarshal the %v: %v", yamlFileName, err)
	}

	var testCases []testCase
	for _, yamlTestCase := range yamlFile {
		testCases = append(testCases, testCase{
			Description: yamlTestCase.Description,
			Script:      readScript(t, yamlTestCase.Script),
			PreState:    readHex32(t, yamlTestCase.PreState),
			Block:       readHex(t, yamlTestCase.Block),
			PostState:   readHex32(t, yamlTestCase.PostState),
			Deposits:    readDeposits(t, yamlTestCase.Deposits),
			Error:       yamlTestCase.Error,
		})
	}

	return testCases
}

func TestExecuteCode(t *testing.T) {
	testCases := readYaml(t, path.Join(testsPath, "_execute_code_test.yaml"))
	for _, test := range testCases {
		postState, deposits, err := executeCode(test.Script, test.PreState, test.Block)
		if err != nil && !strings.HasPrefix(err.Error(), fmt.Sprintf("wasm return error code %v", test.Error)) {
			t.Fatalf("%v\nExecuteCode error: %v \nwait return code: %v", test.Description, err, test.Error)
		}
		if err == nil && test.Error != 0 {
			t.Fatalf("%v\nExecuteCode not return error, but waited error code: %v", test.Description, test.Error)
		}
		if !bytes.Equal(postState[:], test.PostState[:]) {
			t.Fatalf("%v\nExecuteCode incorrect result.\nwait:   %v\nresult: %v", test.Description, test.PostState, postState)
		}

		if len(deposits) > 0 || len(test.Deposits) > 0 {
			if len(deposits) != len(test.Deposits) {
				t.Fatalf("%v\ndeposits count not equal.\nwait:   %v\nresult: %v", test.Description, len(test.Deposits), len(deposits))
			} else {
				for i := 0; i < len(deposits); i++ {
					if !reflect.DeepEqual(deposits[i], test.Deposits[i]) {
						t.Fatalf("%v\ndeposit not equal.\nwait:   %v\nresult: %v", test.Description, test.Deposits[i], deposits[i])
					}
				}
			}
		}
	}
}
