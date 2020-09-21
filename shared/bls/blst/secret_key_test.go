// +build linux,amd64 linux,arm64

package blst_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls/blst"
	"github.com/prysmaticlabs/prysm/shared/bls/iface"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestMarshalUnmarshal(t *testing.T) {
	b := blst.RandKey().Marshal()
	b32 := bytesutil.ToBytes32(b)
	pk, err := blst.SecretKeyFromBytes(b32[:])
	require.NoError(t, err)
	pk2, err := blst.SecretKeyFromBytes(b32[:])
	require.NoError(t, err)
	assert.DeepEqual(t, pk.Marshal(), pk2.Marshal(), "Keys not equal")
}

func TestMultipleSignatureVerificationRepeatedMessages(t *testing.T) {
	pubkeys := make([]iface.PublicKey, 0, 4)
	sigs := make([][]byte, 0, 4)
	var msgs [][32]byte
	for i := 0; i < 4; i++ {
		msg := [32]byte{'h', 'e', 'l', 'l', 'o', byte(i % 2)}
		//t.Errorf("msg: %s", string(msg[:]))
		priv := blst.RandKey()
		pub := priv.PublicKey()
		sig := priv.Sign(msg[:]).Marshal()
		pubkeys = append(pubkeys, pub)
		sigs = append(sigs, sig)
		msgs = append(msgs, msg)
	}
	verify, err := blst.VerifyMultipleSignatures(append(sigs, sigs...), append(msgs, msgs...), append(pubkeys, pubkeys...))
	assert.NoError(t, err, "Signature did not verify")
	assert.Equal(t, true, verify, "Signature did not verify")
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
			err:   errors.New("could not unmarshal bytes into secret key"),
		},
		{
			name:  "Good",
			input: []byte{0x25, 0x29, 0x5f, 0x0d, 0x1d, 0x59, 0x2a, 0x90, 0xb3, 0x33, 0xe2, 0x6e, 0x85, 0x14, 0x97, 0x08, 0x20, 0x8e, 0x9f, 0x8e, 0x8b, 0xc1, 0x8f, 0x6c, 0x77, 0xbd, 0x62, 0xf8, 0xad, 0x7a, 0x68, 0x66},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res, err := blst.SecretKeyFromBytes(test.input)
			if test.err != nil {
				assert.NotEqual(t, nil, err, "No error returned")
				assert.ErrorContains(t, test.err.Error(), err, "Unexpected error returned")
			} else {
				assert.NoError(t, err)
				assert.DeepEqual(t, 0, bytes.Compare(res.Marshal(), test.input))
			}
		})
	}
}

func TestSerialize(t *testing.T) {
	rk := blst.RandKey()
	b := rk.Marshal()

	_, err := blst.SecretKeyFromBytes(b)
	assert.NoError(t, err)
}
