package v2_test

import (
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
)

var _ = v2keymanager.IKeymanager(&direct.Keymanager{})
