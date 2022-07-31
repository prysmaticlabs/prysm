//go:build linux

package journald

import (
	"github.com/wercker/journalhook"
)

//Enable enables the journald  logrus hook
func Enable() error {
	journalhook.Enable()
	return nil
}
