/**
Package from the official Github CLI https://github.com/cli/cli/blob/f30bc5bc64f9c3a839e39713adab48790264119c/pkg/browser/browser.go
All rights reserved to the package authors, respectively. MIT License. See https://github.com/cli/cli/blob/trunk/LICENSE
*/
package browser

import (
	"os"
	"os/exec"
	"strings"

	"github.com/google/shlex"
)

// ForOS produces an exec.Cmd to open the web browser for different OS
func ForOS(goos, url string) *exec.Cmd {
	exe := "open"
	var args []string
	switch goos {
	case "darwin":
		args = append(args, url)
	case "windows":
		exe = "cmd"
		r := strings.NewReplacer("&", "^&")
		args = append(args, "/c", "start", r.Replace(url))
	default:
		exe = "xdg-open"
		args = append(args, url)
	}

	cmd := exec.Command(exe, args...)
	cmd.Stderr = os.Stderr
	return cmd
}

// FromLauncher parses the launcher string based on shell splitting rules
func FromLauncher(launcher, url string) (*exec.Cmd, error) {
	args, err := shlex.Split(launcher)
	if err != nil {
		return nil, err
	}

	args = append(args, url)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stderr = os.Stderr
	return cmd, nil
}
