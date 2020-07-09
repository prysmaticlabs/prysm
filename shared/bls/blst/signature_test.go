package blst_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls/blst"
	"github.com/prysmaticlabs/prysm/shared/bls/iface"
)

func TestSignVerify(t *testing.T) {
	priv := blst.RandKey()
	pub := priv.PublicKey()
	msg := []byte("hello")
	sig := priv.Sign(msg)
	if !sig.Verify(pub, msg) {
		t.Error("Signature did not verify")
	}
}

func TestAggregateVerify(t *testing.T) {
	pubkeys := make([]iface.PublicKey, 0, 100)
	sigs := make([]iface.Signature, 0, 100)
	var msgs [][32]byte
	for i := 0; i < 100; i++ {
		msg := [32]byte{'h', 'e', 'l', 'l', 'o', byte(i)}
		priv := blst.RandKey()
		pub := priv.PublicKey()
		sig := priv.Sign(msg[:])
		pubkeys = append(pubkeys, pub)
		sigs = append(sigs, sig)
		msgs = append(msgs, msg)
	}
	aggSig := blst.Aggregate(sigs)
	if !aggSig.AggregateVerify(pubkeys, msgs) {
		t.Error("Signature did not verify")
	}
}

func TestFastAggregateVerify(t *testing.T) {
	pubkeys := make([]iface.PublicKey, 0, 100)
	sigs := make([]iface.Signature, 0, 100)
	msg := [32]byte{'h', 'e', 'l', 'l', 'o'}
	for i := 0; i < 100; i++ {
		priv := blst.RandKey()
		pub := priv.PublicKey()
		sig := priv.Sign(msg[:])
		pubkeys = append(pubkeys, pub)
		sigs = append(sigs, sig)
	}
	aggSig := blst.AggregateSignatures(sigs)
	if !aggSig.FastAggregateVerify(pubkeys, msg) {
		t.Error("Signature did not verify")
	}
}
/*
func TestMultipleSignatureVerification(t *testing.T) {
	pubkeys := make([]iface.PublicKey, 0, 100)
	sigs := make([]iface.Signature, 0, 100)
	var msgs [][32]byte
	for i := 0; i < 100; i++ {
		msg := [32]byte{'h', 'e', 'l', 'l', 'o', byte(i)}
		priv := blst.RandKey()
		pub := priv.PublicKey()
		sig := priv.Sign(msg[:])
		pubkeys = append(pubkeys, pub)
		sigs = append(sigs, sig)
		msgs = append(msgs, msg)
	}
	if verify, err := blst.VerifyMultipleSignatures(sigs, msgs, pubkeys); !verify || err != nil {
		t.Errorf("Signature did not verify: %v and err %v", verify, err)
	}
}

func TestMultipleSignatureVerification_FailsCorrectly(t *testing.T) {
	pubkeys := make([]iface.PublicKey, 0, 100)
	sigs := make([]iface.Signature, 0, 100)
	var msgs [][32]byte
	for i := 0; i < 100; i++ {
		msg := [32]byte{'h', 'e', 'l', 'l', 'o', byte(i)}
		priv := blst.RandKey()
		pub := priv.PublicKey()
		sig := priv.Sign(msg[:])
		pubkeys = append(pubkeys, pub)
		sigs = append(sigs, sig)
		msgs = append(msgs, msg)
	}
	// We mess with the last 2 signatures, where we modify their values
	// such that they wqould not fail in aggregate signature verification.
	lastSig := sigs[len(sigs)-1]
	secondLastSig := sigs[len(sigs)-2]
	// Convert to bls object
	rawSig := new(bls12.Sign)
	if err := rawSig.Deserialize(secondLastSig.Marshal()); err != nil {
		t.Fatal(err)
	}

	rawSig2 := new(bls12.Sign)
	if err := rawSig2.Deserialize(lastSig.Marshal()); err != nil {
		t.Fatal(err)
	}

	// set random field prime value
	fprime := new(bls12.Fp)
	fprime.SetInt64(100)

	// set random field prime value.
	fprime2 := new(bls12.Fp)
	fprime2.SetInt64(50)

	// make a combined fp2 object.
	fp2 := new(bls12.Fp2)
	fp2.D = [2]bls12.Fp{*fprime, *fprime2}

	g2Point := new(bls12.G2)
	if err := bls12.MapToG2(g2Point, fp2); err != nil {
		t.Fatal(err)
	}

	// We now add/subtract the respective g2 points by a fixed
	// value. This would cause singluar verification to fail but
	// not aggregate verification.
	firstG2 := bls12.CastFromSign(rawSig)
	secondG2 := bls12.CastFromSign(rawSig2)
	bls12.G2Add(firstG2, firstG2, g2Point)
	bls12.G2Sub(secondG2, secondG2, g2Point)

	lastSig, err := blst.SignatureFromBytes(rawSig.Serialize())
	if err != nil {
		t.Fatal(err)
	}
	secondLastSig, err = blst.SignatureFromBytes(rawSig2.Serialize())
	if err != nil {
		t.Fatal(err)
	}
	sigs[len(sigs)-1] = lastSig
	sigs[len(sigs)-2] = secondLastSig

	// This method is expected to pass, as it would not
	// be able to detect bad signatures
	aggSig := blst.AggregateSignatures(sigs)
	if !aggSig.AggregateVerify(pubkeys, msgs) {
		t.Error("Signature did not verify")
	}
	// This method would be expected to fail.
	if verify, err := blst.VerifyMultipleSignatures(sigs, msgs, pubkeys); verify || err != nil {
		t.Errorf("Signature verified when it was not supposed to: %v and err %v", verify, err)
	}
}
*/
func TestFastAggregateVerify_ReturnsFalseOnEmptyPubKeyList(t *testing.T) {
	var pubkeys []iface.PublicKey
	msg := [32]byte{'h', 'e', 'l', 'l', 'o'}

	aggSig := blst.NewAggregateSignature()
	if aggSig.FastAggregateVerify(pubkeys, msg) != false {
		t.Error("Expected FastAggregateVerify to return false with empty input " +
			"of public keys.")
	}
}

func TestSignatureFromBytes(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		err   error
	}{
		{
			name: "Nil",
			err:  errors.New("signature must be 96 bytes"),
		},
		{
			name:  "Empty",
			input: []byte{},
			err:   errors.New("signature must be 96 bytes"),
		},
		{
			name:  "Short",
			input: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			err:   errors.New("signature must be 96 bytes"),
		},
		{
			name:  "Long",
			input: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			err:   errors.New("signature must be 96 bytes"),
		},
		{
			name:  "Bad",
			input: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			err:   errors.New("could not unmarshal bytes into signature: err blsSignatureDeserialize 000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			name:  "Good",
			input: []byte{0xab, 0xb0, 0x12, 0x4c, 0x75, 0x74, 0xf2, 0x81, 0xa2, 0x93, 0xf4, 0x18, 0x5c, 0xad, 0x3c, 0xb2, 0x26, 0x81, 0xd5, 0x20, 0x91, 0x7c, 0xe4, 0x66, 0x65, 0x24, 0x3e, 0xac, 0xb0, 0x51, 0x00, 0x0d, 0x8b, 0xac, 0xf7, 0x5e, 0x14, 0x51, 0x87, 0x0c, 0xa6, 0xb3, 0xb9, 0xe6, 0xc9, 0xd4, 0x1a, 0x7b, 0x02, 0xea, 0xd2, 0x68, 0x5a, 0x84, 0x18, 0x8a, 0x4f, 0xaf, 0xd3, 0x82, 0x5d, 0xaf, 0x6a, 0x98, 0x96, 0x25, 0xd7, 0x19, 0xcc, 0xd2, 0xd8, 0x3a, 0x40, 0x10, 0x1f, 0x4a, 0x45, 0x3f, 0xca, 0x62, 0x87, 0x8c, 0x89, 0x0e, 0xca, 0x62, 0x23, 0x63, 0xf9, 0xdd, 0xb8, 0xf3, 0x67, 0xa9, 0x1e, 0x84},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res, err := blst.SignatureFromBytes(test.input)
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
