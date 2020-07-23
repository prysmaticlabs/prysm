package v2_test

import (
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/remote"
)

var (
	_ = v2keymanager.IKeymanager(&direct.Keymanager{})
	_ = v2keymanager.IKeymanager(&derived.Keymanager{})
	_ = v2keymanager.IKeymanager(&remote.Keymanager{})
)
