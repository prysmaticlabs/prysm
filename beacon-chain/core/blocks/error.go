package blocks

import "github.com/pkg/errors"

var errNilSignedWithdrawalMessage = errors.New("nil SignedBLSToExecutionChange message")
var errNilWithdrawalMessage = errors.New("nil BLSToExecutionChange message")
var errInvalidBLSPrefix = errors.New("invalid BLS withdrawal credential prefix")
var errInvalidWithdrawalCredentials = errors.New("withdrawal credentials do not match")
