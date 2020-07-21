package keystore

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/pborman/uuid"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestMarshalAndUnmarshal(t *testing.T) {
	testID := uuid.NewRandom()
	blsKey := bls.RandKey()

	key := &Key{
		ID:        testID,
		SecretKey: blsKey,
		PublicKey: blsKey.PublicKey(),
	}
	marshalledObject, err := key.MarshalJSON()
	require.NoError(t, err)
	newKey := &Key{
		ID:        []byte{},
		SecretKey: blsKey,
		PublicKey: blsKey.PublicKey(),
	}

	err = newKey.UnmarshalJSON(marshalledObject)
	require.NoError(t, err)
	require.Equal(t, true, bytes.Equal(newKey.ID, testID))
}

func TestStoreRandomKey(t *testing.T) {
	tempDir, teardown := setupTempKeystoreDir(t)
	defer teardown()
	ks := &Store{
		keysDirPath: tempDir,
		scryptN:     LightScryptN,
		scryptP:     LightScryptP,
	}
	require.NoError(t, storeNewRandomKey(ks, "password"))
}

func TestNewKeyFromBLS(t *testing.T) {
	b := []byte("hi")
	b32 := bytesutil.ToBytes32(b)
	blskey, err := bls.SecretKeyFromBytes(b32[:])
	require.NoError(t, err)
	key, err := NewKeyFromBLS(blskey)
	require.NoError(t, err)

	expected := blskey.Marshal()
	require.Equal(t, true, bytes.Equal(expected, key.SecretKey.Marshal()))
	_, err = NewKey()
	require.NoError(t, err)
}

func TestWriteFile(t *testing.T) {
	tempDir, teardown := setupTempKeystoreDir(t)
	defer teardown()

	testKeystore := []byte{'t', 'e', 's', 't'}

	err := writeKeyFile(tempDir, testKeystore)
	require.NoError(t, err)

	keystore, err := ioutil.ReadFile(tempDir)
	require.NoError(t, err)
	require.Equal(t, true, bytes.Equal(keystore, testKeystore))
}
