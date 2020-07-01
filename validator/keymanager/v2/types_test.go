package v2

import "github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"

var _ = IKeymanager(&direct.Keymanager{})
