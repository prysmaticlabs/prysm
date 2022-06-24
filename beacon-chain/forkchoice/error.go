package forkchoice

import "github.com/pkg/errors"

var ErrUnknownCommonAncestor = errors.New("unknown common ancestor")
var ErrNilNode = errors.New("nil node")
