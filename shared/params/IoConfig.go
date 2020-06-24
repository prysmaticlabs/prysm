package params

import "os"

// IoConfig defines the shared io parameters.
type IoConfig struct {
	FilePermission os.FileMode
}

var defaultIoConfig = &IoConfig{
	FilePermission: 0600, //-rw------- Read and Write permissions for user
}

// BeaconNetworkConfig returns the current network config for
// the beacon chain.
func BeaconIoConfig() *IoConfig {
	return defaultIoConfig
}
