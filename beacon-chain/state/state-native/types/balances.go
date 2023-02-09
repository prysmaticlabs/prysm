package types

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type ValidatorBalancesReadOnly struct {
	value []uint64
}

func NewValidatorBalances(value []uint64) *ValidatorBalancesReadOnly {
	return &ValidatorBalancesReadOnly{value: value}
}

func (b *ValidatorBalancesReadOnly) Value() []uint64 {
	res := make([]uint64, len(b.value))
	copy(res, b.value)
	log.Warn("copy balances")
	return res
}

func (b *ValidatorBalancesReadOnly) At(index uint64) (uint64, error) {
	if index > uint64(len(b.value)) {
		return 0, errors.New("index out of bounds")
	}
	return b.value[index], nil
}

func (b *ValidatorBalancesReadOnly) Len() int {
	return len(b.value)
}
