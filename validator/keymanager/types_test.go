package keymanager_test

import (
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/deprecatedderived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/imported"
	"github.com/prysmaticlabs/prysm/validator/keymanager/remote"
)

var (
	_ = keymanager.IKeymanager(&imported.Keymanager{})
	_ = keymanager.IKeymanager(&deprecatedderived.Keymanager{})
	_ = keymanager.IKeymanager(&remote.Keymanager{})
)
