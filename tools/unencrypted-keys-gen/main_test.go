package main

import (
	"bytes"
	"encoding/json"
	"github.com/prysmaticlabs/prysm/tools/unencrypted-keys-gen/keygen"
	"reflect"
	"testing"
)

func TestSavesUnencryptedKeys(t *testing.T) {
	keys := 2
	numKeys = &keys
	ctnr := generateUnencryptedKeys(0 /* start index */)
	buf := new(bytes.Buffer)
	if err := keygen.SaveUnencryptedKeysToFile(buf, ctnr); err != nil {
		t.Fatal(err)
	}
	enc := buf.Bytes()
	dec := &keygen.UnencryptedKeysContainer{}
	if err := json.Unmarshal(enc, dec); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ctnr, dec) {
		t.Errorf("Wanted %v, received %v", ctnr, dec)
	}
}
