package types

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type ValidatorBalances struct {
	value []uint64
}

func NewValidatorBalances(value []uint64) *ValidatorBalances {
	return &ValidatorBalances{value: value}
}

func (b *ValidatorBalances) Value() []uint64 {
	res := make([]uint64, len(b.value))
	copy(res, b.value)
	log.Warn("copy balances value")
	//debug.PrintStack()
	return res
}

func (b *ValidatorBalances) At(index uint64) (uint64, error) {
	if index > uint64(len(b.value)) {
		return 0, errors.New("index out of bounds")
	}
	return b.value[index], nil
}

func (b *ValidatorBalances) Len() int {
	return len(b.value)
}
