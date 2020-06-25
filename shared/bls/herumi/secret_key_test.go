package herumi_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls/herumi"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

func TestMarshalUnmarshal(t *testing.T) {
	b := herumi.RandKey().Marshal()
	b32 := bytesutil.ToBytes32(b)
	pk, err := herumi.SecretKeyFromBytes(b32[:])
	if err != nil {
		t.Fatal(err)
	}
	pk2, err := herumi.SecretKeyFromBytes(b32[:])
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(pk.Marshal(), pk2.Marshal()) {
		t.Errorf("Keys not equal, received %#x == %#x", pk.Marshal(), pk2.Marshal())
	}
}

func TestSecretKeyFromBytes(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		err   error
	}{
		{
			name: "Nil",
			err:  errors.New("secret key must be 32 bytes"),
		},
		{
			name:  "Empty",
			input: []byte{},
			err:   errors.New("secret key must be 32 bytes"),
		},
		{
			name:  "Short",
			input: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			err:   errors.New("secret key must be 32 bytes"),
		},
		{
			name:  "Long",
			input: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			err:   errors.New("secret key must be 32 bytes"),
		},
		{
			name:  "Bad",
			input: []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			err:   errors.New("could not unmarshal bytes into secret key: err blsSecretKeyDeserialize ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
		},
		{
			name:  "Good",
			input: []byte{0x25, 0x29, 0x5f, 0x0d, 0x1d, 0x59, 0x2a, 0x90, 0xb3, 0x33, 0xe2, 0x6e, 0x85, 0x14, 0x97, 0x08, 0x20, 0x8e, 0x9f, 0x8e, 0x8b, 0xc1, 0x8f, 0x6c, 0x77, 0xbd, 0x62, 0xf8, 0xad, 0x7a, 0x68, 0x66},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res, err := herumi.SecretKeyFromBytes(test.input)
			if test.err != nil {
				if err == nil {
					t.Errorf("No error returned: expected %v", test.err)
				} else if test.err.Error() != err.Error() {
					t.Errorf("Unexpected error returned: expected %v, received %v", test.err, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error returned: %v", err)
				} else {
					if bytes.Compare(res.Marshal(), test.input) != 0 {
						t.Errorf("Unexpected result: expected %x, received %x", test.input, res.Marshal())
					}
				}
			}

		})
	}
}

func TestSerialize(t *testing.T) {
	rk := herumi.RandKey()
	b := rk.Marshal()

	if _, err := herumi.SecretKeyFromBytes(b); err != nil {
		t.Error(err)
	}
}
