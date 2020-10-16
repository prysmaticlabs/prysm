package keymanager_test

import (
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/direct"
	"github.com/prysmaticlabs/prysm/validator/keymanager/remote"
)

var (
	_ = keymanager.IKeymanager(&direct.Keymanager{})
	_ = keymanager.IKeymanager(&derived.Keymanager{})
	_ = keymanager.IKeymanager(&remote.Keymanager{})
)
