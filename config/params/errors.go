package params

import "github.com/pkg/errors"

// ErrVersionNotFound indicates the config package couldn't determine the version for an epoch using the fork schedule.
var ErrVersionNotFound = errors.New("could not find an entry in the fork schedule")
