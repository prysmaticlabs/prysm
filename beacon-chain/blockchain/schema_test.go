package blockchain

import (
	"bytes"
	"testing"
)

func TestBlockKeys(t *testing.T) {

	testhash := [32]byte{1, 2, 4, 5, 6, 7, 8, 9, 10}
	testkey := append(blockPrefix, testhash[:]...)
	generatedKey := blockKey(testhash)

	if !bytes.Equal(testkey, generatedKey) {
		t.Errorf("block keys are not the same %v, %v", testkey, generatedKey)
	}

	testslotnumber := uint64(4)
	expectedKey := append(canonicalPrefix, encodeSlotNumber(testslotnumber)...)
	generatedkey := canonicalBlockKey(testslotnumber)

	if !bytes.Equal(generatedkey, expectedKey) {
		t.Errorf("expected and generated canonical keys are not equal %v, %v", expectedKey, generatedKey)
	}

}
