package prereq

import (
	"context"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type platform struct {
	os           string
	arch         string
	majorVersion int
	minorVersion int
}

func getSupportedPlatforms() []platform {
	return []platform{
		{os: "linux", arch: "amd64"},
		{os: "linux", arch: "arm64"},
		{os: "darwin", arch: "amd64", majorVersion: 10, minorVersion: 14},
		{os: "windows", arch: "amd64"},
	}
}

// parseVersion takes a string and splits it using sep separator, and outputs a slice of integers
// corresponding to version numbers.  If it cannot find num level of versions, it returns an error
func parseVersion(input string, num int, sep string) ([]int, error) {
	var version = make([]int, num)
	components := strings.Split(input, sep)
	if len(components) < num {
		return nil, errors.New("insufficient information about version")
	}
	for i := range version {
		var err error
		version[i], err = strconv.Atoi(components[i])
		if err != nil {
			return nil, errors.Wrap(err, "error during conversion")
		}
	}
	return version, nil
}

// meetsMinPlatformReqs returns true if the runtime matches any on the list of supported platforms
func meetsMinPlatformReqs(ctx context.Context) (bool, error) {
	okPlatforms := getSupportedPlatforms()
	runtimeOS := runtime.GOOS
	runtimeArch := runtime.GOARCH
	for _, platform := range okPlatforms {
		if runtimeOS == platform.os && runtimeArch == platform.arch {
			// If MacOS we make sure it meets the minimum version cutoff
			if runtimeOS == "darwin" {
				versionBytes, err := exec.CommandContext(ctx, "uname", "-r").Output()
				if err != nil {
					return false, errors.Wrap(err, "error obtaining MacOS version")
				}
				version, err := parseVersion(string(versionBytes), 2, ".")
				if err != nil {
					return false, errors.Wrap(err, "error parsing version")
				}
				if version[0] != platform.majorVersion {
					return version[0] > platform.majorVersion, nil
				}
				if version[1] < platform.minorVersion {
					return false, nil
				}
				return true, nil
			}
			// Otherwise we have a match between runtime and our list of accepted platforms
			return true, nil
		}
	}
	return false, nil
}

// WarnIfNotSupported warns if the user's platform is not supported
func WarnIfNotSupported(ctx context.Context) error {
	supported, err := meetsMinPlatformReqs(ctx)
	if err != nil {
		return err
	}
	if !supported {
		log.Warn("This platform is not supported. The following platforms are supported: Linux/AMD64," +
			" Linux/ARM64, Mac OS X/AMD64 (10.14+ only), and Windows/AMD64")
	}
	return nil
}
