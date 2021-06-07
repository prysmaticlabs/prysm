package synccommittee

import "github.com/pkg/errors"

var (
	nilSignatureErr    = errors.New("sync committee signature is nil")
	nilContributionErr = errors.New("sync committee contribution is nil")
)
