package keymanager_test

import (
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/imported"
)

var (
	_ = keymanager.IKeymanager(&imported.Keymanager{})
	//_ = keymanager.IKeymanager(&remote.Keymanager{})
)
