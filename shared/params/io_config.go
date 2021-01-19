package params

import (
	"os"
	"time"
)

// IoConfig defines the shared io parameters.
type IoConfig struct {
	ReadWritePermissions        os.FileMode
	ReadWriteExecutePermissions os.FileMode
	BoltTimeout                 time.Duration
}

var defaultIoConfig = &IoConfig{
	ReadWritePermissions:        0600,            //-rw------- Read and Write permissions for user
	ReadWriteExecutePermissions: 0700,            //-rwx------ Read Write and Execute (traverse) permissions for user
	BoltTimeout:                 1 * time.Second, // 1 second for the bolt DB to timeout.
}

// BeaconIoConfig returns the current io config for
// the beacon chain.
func BeaconIoConfig() *IoConfig {
	return defaultIoConfig
}
