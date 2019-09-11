package main

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

func TestSavesUnencryptedKeys(t *testing.T) {
	ctnr := generateUnencryptedKeys()
	buf := new(bytes.Buffer)
	if err := SaveUnencryptedKeysToFile(buf, ctnr); err != nil {
		t.Fatal(err)
	}
	enc := buf.Bytes()
	dec := &UnencryptedKeysContainer{}
	if err := json.Unmarshal(enc, dec); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ctnr, dec) {
		t.Errorf("Wanted %v, received %v", ctnr, dec)
	}
}
