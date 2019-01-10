package version

import (
	"fmt"
)

// The value of these vars are set through linker options.
var gitCommit = "Local build"
var buildDate = "Moments ago"

// GetVersion returns the version string of this build.
func GetVersion() string {
	return fmt.Sprintf("Git commit: %s. Built at: %s", gitCommit, buildDate)
}
