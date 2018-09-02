package blockchain

import (
	"bytes"
	"testing"
)

func TestEncodeAndDecodeSlotNumber(t *testing.T) {
	testValues := []uint64{20, 200, 88, 90, 19, 8029, 36209, 202928}
	for _, val := range testValues {

		encodednum := encodeSlotNumber(val)
		expectednumber := decodeSlotNumber(encodednum)

		if val != expectednumber {
			t.Errorf("encoding leads to different expected and generated slotnumber %v, %v", val, expectednumber)
		}
	}
}

func TestBlockKeys(t *testing.T) {
	var testhash [32]byte
	testhash = [32]byte{1, 2, 4, 5, 6, 7, 8, 9, 10}
	testkey := append(blockPrefix, testhash[:]...)
	generatedKey := blockKey(testhash)

	if !bytes.Equal(testkey, generatedKey) {
		t.Errorf("block keys are not the same %v, %v", testkey, generatedKey)
	}

	testslotnumber := uint64(4)
	expectedKey := append(canonicalPrefix, []byte{0, 0, 0, 0, 0, 0, 0, byte(testslotnumber)}...)
	generatedkey := canonicalBlockKey(testslotnumber)

	if !bytes.Equal(generatedkey, expectedKey) {
		t.Errorf("expected and generated canonical keys are not equal %v, %v", expectedKey, generatedKey)
	}

}
