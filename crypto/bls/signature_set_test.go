package bls

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func TestCopySignatureSet(t *testing.T) {
	t.Run("blst", func(t *testing.T) {
		key, err := RandKey()
		assert.NoError(t, err)
		key2, err := RandKey()
		assert.NoError(t, err)
		key3, err := RandKey()
		assert.NoError(t, err)

		message := [32]byte{'C', 'D'}
		message2 := [32]byte{'E', 'F'}
		message3 := [32]byte{'H', 'I'}

		sig := key.Sign(message[:])
		sig2 := key2.Sign(message2[:])
		sig3 := key3.Sign(message3[:])

		set := &SignatureSet{
			Signatures: [][]byte{sig.Marshal()},
			PublicKeys: []PublicKey{key.PublicKey()},
			Messages:   [][32]byte{message},
		}
		set2 := &SignatureSet{
			Signatures: [][]byte{sig2.Marshal()},
			PublicKeys: []PublicKey{key.PublicKey()},
			Messages:   [][32]byte{message},
		}
		set3 := &SignatureSet{
			Signatures: [][]byte{sig3.Marshal()},
			PublicKeys: []PublicKey{key.PublicKey()},
			Messages:   [][32]byte{message},
		}
		aggSet := set.Join(set2).Join(set3)
		aggSet2 := aggSet.Copy()

		assert.DeepEqual(t, aggSet, aggSet2)
	})
}
