package builder

import "github.com/pkg/errors"

var (
	ErrNotRunning = errors.New("builder is not running")
)
