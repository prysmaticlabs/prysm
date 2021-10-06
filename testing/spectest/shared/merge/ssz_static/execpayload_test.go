package ssz_static

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"github.com/prysmaticlabs/prysm/testing/spectest/utils"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/golang/snappy"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

type SSZValue struct {
	Message json.RawMessage `json:"message"`
	Signature string `json:"signature"`// hex encoded '0x...'
}

func TestExecPayload(t *testing.T) {
	tc := &TestCase{
		path: "testdata/ssz_random/case_1",
	}
	ssb, err := tc.MarshaledBytes()
	if err != nil {
		t.Error(err)
	}

	block := &ethpb.BeaconBlockMerge{}
	err = block.UnmarshalSSZ(ssb)
	if err != nil {
		t.Error(err)
	}
	bb, err := block.MarshalSSZ()
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(bb, ssb) {
		t.Error("Expected marshaled bytes to match fixture")
	}
	htr, err := block.HashTreeRoot()
	if err != nil {
		t.Error(err)
	}
	rb, err := tc.RootBytes()
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(htr[:], rb[:]) {
		t.Error("HTR was not equal to expected value")
	}
}

func getSSZSnappy(path string) ([]byte, error) {
		fh, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer fh.Close()
		buf := bytes.NewBuffer(nil)
		_, err = buf.ReadFrom(fh)
		return snappy.Decode(nil, buf.Bytes())
}

type TestCase struct {
	path string
	config string
	phase string
	typeName string
	caseId string
}

func (tc *TestCase) MarshaledBytes() ([]byte, error) {
	fh, err := os.Open(path.Join(tc.path, "serialized.ssz_snappy"))
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	buf := bytes.NewBuffer(nil)
	_, err = buf.ReadFrom(fh)
	return snappy.Decode(nil, buf.Bytes())
}

func (tc *TestCase) Value() (*SSZValue, error) {
	fh, err := os.Open(path.Join(tc.path, "value.yaml"))
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	d := json.NewDecoder(fh)
	v := &SSZValue{}
	err = d.Decode(v)
	return v, err
}

func (tc *TestCase) Roots() (*SSZRoots, error) {
	fh, err := os.Open(path.Join(tc.path, "roots.yaml"))
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	//d := json.NewDecoder(fh)
	b, err := ioutil.ReadAll(fh)
	if err != nil {
		return nil, err
	}
	r := &SSZRoots{}
	err = utils.UnmarshalYaml(b, r)
	return r, err
}

func (tc *TestCase) RootBytes() ([32]byte, error) {
	var b [32]byte
	r, err := tc.Roots()
	if err != nil {
		return b, err
	}
	rootBytes, err := hex.DecodeString(r.Root[2:])
	if err != nil {
		return b, err
	}
	copy(b[:], rootBytes[:])
	return b, nil
}