package mdns

import "log"

// DisableLogging disables all log messages
var DisableLogging bool

func logf(format string, args ...interface{}) {
	if DisableLogging {
		return
	}
	log.Printf(format, args...)
}
