package version

import (
	"fmt"
)

// The value of these vars are set through linker options.
var GitCommit string = "Local build"
var BuildDate string = "Moments ago"

// GetVersion returns the version string of this build.
func GetVersion() string {
	return fmt.Sprintf("Git commit: %s. Built at: %s", GitCommit, BuildDate)
}
