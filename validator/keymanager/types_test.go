package keymanager_test

import (
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/local"
	"github.com/prysmaticlabs/prysm/validator/keymanager/remote"
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
)
