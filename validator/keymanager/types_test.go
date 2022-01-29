package keymanager_test

import (
	"encoding/json"
	"testing"

	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/imported"
	"github.com/prysmaticlabs/prysm/validator/keymanager/remote"
)

var (
	_ = keymanager.IKeymanager(&imported.Keymanager{})
	_ = keymanager.IKeymanager(&derived.Keymanager{})
	_ = keymanager.IKeymanager(&remote.Keymanager{})

	// More granular assertions.
	_ = keymanager.KeysFetcher(&imported.Keymanager{})
	_ = keymanager.KeysFetcher(&derived.Keymanager{})
	_ = keymanager.Importer(&imported.Keymanager{})
	_ = keymanager.Importer(&derived.Keymanager{})
	_ = keymanager.Deleter(&imported.Keymanager{})
	_ = keymanager.Deleter(&derived.Keymanager{})
)

func TestKeystoreContainsPath(t *testing.T) {
	keystore := keymanager.Keystore{}
	encoded, err := json.Marshal(keystore)
	want := "{\"crypto\":null,\"uuid\":\"\",\"pubkey\":\"\",\"version\":0,\"name\":\"\",\"path\":\"\"}"

	require.NoError(t, err, "Unexpected error marshalling keystore")
	require.Equal(t, want, string(encoded))
}
