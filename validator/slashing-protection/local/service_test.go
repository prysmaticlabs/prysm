package local_test

import (
	"github.com/prysmaticlabs/prysm/validator/slashing-protection"
	"github.com/prysmaticlabs/prysm/validator/slashing-protection/local"
)

var (
	_ = slashingprotection.Protector(&local.Service{})
)
