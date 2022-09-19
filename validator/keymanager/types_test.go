package keymanager_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager/derived"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager/local"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager/remote"
	remoteweb3signer "github.com/prysmaticlabs/prysm/v3/validator/keymanager/remote-web3signer"
)

var (
	_ = keymanager.IKeymanager(&local.Keymanager{})
	_ = keymanager.IKeymanager(&derived.Keymanager{})
	_ = keymanager.IKeymanager(&remote.Keymanager{})

	// More granular assertions.
	_ = keymanager.KeysFetcher(&local.Keymanager{})
	_ = keymanager.KeysFetcher(&derived.Keymanager{})
	_ = keymanager.Importer(&local.Keymanager{})
	_ = keymanager.Importer(&derived.Keymanager{})
	_ = keymanager.Deleter(&local.Keymanager{})
	_ = keymanager.Deleter(&derived.Keymanager{})

	_ = keymanager.PublicKeyAdder(&remoteweb3signer.Keymanager{})
	_ = keymanager.PublicKeyDeleter(&remoteweb3signer.Keymanager{})
)

func TestKeystoreContainsPath(t *testing.T) {
	keystore := keymanager.Keystore{}
	encoded, err := json.Marshal(keystore)

	require.NoError(t, err, "Unexpected error marshalling keystore")
	assert.Equal(t, true, strings.Contains(string(encoded), "path"))
}
