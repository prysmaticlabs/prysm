package blocks

import "github.com/pkg/errors"

var errNilSignedWithdrawalMessage = errors.New("nil SignedBLSToExecutionChange message")
var errNilWithdrawalMessage = errors.New("nil BLSToExecutionChange message")
var errInvalidBLSPrefix = errors.New("withdrawal credential prefix is not a BLS prefix")
var errInvalidWithdrawalCredentials = errors.New("withdrawal credentials do not match")
var errInvalidWithdrawalIndex = errors.New("invalid withdrawal index")
var errInvalidValidatorIndex = errors.New("invalid validator index")
var errInvalidWithdrawalAmount = errors.New("invalid withdrawal amount")
var errInvalidExecutionAddress = errors.New("invalid execution address")
var errInvalidWithdrawalNumber = errors.New("invalid number of withdrawals")
